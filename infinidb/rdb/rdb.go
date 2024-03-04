package rdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"
)

var regItemNotFound = regexp.MustCompile("item ([^ ]+) not found")

// An error indicating an item doesn't exist in the database
type ErrItemNotFound struct {
	Item string
}

func (e ErrItemNotFound) Error() string {
	return fmt.Sprintf("item %s not found", e.Item)
}

// Represents a recipe with two inputs and one result.
type Recipe struct {
	// First recipe source
	SrcA string `json:"src_a"`
	// Second recipe source
	SrcB string `json:"src_b"`
	// Recipe output
	Result string `json:"result"`
}

// Like 'Recipe' but without the result.
type RecipeSources struct {
	SrcA string
	SrcB string
}

// Interface for interacting with the recipe and item database
type RecipeDb interface {
	RecipeJsonTree(ctx context.Context, item string, pretty bool) (json.RawMessage, error)
	ListRecipes(ctx context.Context, item string) ([]RecipeSources, error)
	ItemsLike(ctx context.Context, query string, max uint) ([]string, error)
	// Given the item srcA and the list of second items, returns items from listSrcB which
	// do NOT have a known combination with srcA.
	GetMissingRecipes(ctx context.Context, srcA string, listSrcB []string) ([]string, error)
	AddRecipe(ctx context.Context, recipe Recipe) error
}

// RecipeDb implementation
type recipeDbImpl struct {
	c *pgx.Conn
	// Regex to make ItemsLike doesn't have extra characters
	likeReg *regexp.Regexp
}

func NewRecipeDb(conn *pgx.Conn) RecipeDb {
	return &recipeDbImpl{
		c:       conn,
		likeReg: regexp.MustCompile("[^A-Za-z0-9 ]"),
	}
}

func (me *recipeDbImpl) RecipeJsonTree(ctx context.Context, item string, pretty bool) (tree json.RawMessage, err error) {
	// Try to get the tree
	if err = me.c.QueryRow(ctx, `select v1_find_recipe($1)`, item).Scan(&tree); err != nil {
		return
	}
	if pretty {
		// Try to pretty-print
		prettyBuf := new(bytes.Buffer)
		json.Indent(prettyBuf, tree, "", "  ")
		tree = prettyBuf.Bytes()
	}
	return
}

func (me *recipeDbImpl) ListRecipes(ctx context.Context, item string) ([]RecipeSources, error) {
	const QUERY = `
	select r.name_a, r.name_b
	from recipe_ext r
	where r.dst = (select id from v1_get_item($1))
	`
	rows, _ := me.c.Query(ctx, QUERY, item)
	return pgx.CollectRows(rows, pgx.RowToStructByPos[RecipeSources])
}

func (me *recipeDbImpl) ItemsLike(ctx context.Context, q string, max uint) ([]string, error) {
	qStr := "%" + me.likeReg.ReplaceAllString(q, "") + "%"
	rows, _ := me.c.Query(ctx, "select name from item where lower(name) like lower($1) limit $2", qStr, max)
	if names, err := pgx.CollectRows(rows, pgx.RowTo[string]); err != nil {
		return nil, err
	} else {
		return names, nil
	}
}

func (me *recipeDbImpl) AddRecipe(ctx context.Context, params Recipe) error {
	tx, err := me.c.Begin(ctx)
	defer tx.Rollback(ctx)
	if err != nil {
		return err
	}
	var recipeId int
	row := tx.QueryRow(ctx, "select v1_add_recipe($1,$2,$3)", params.SrcA, params.SrcB, params.Result)
	if err := row.Scan(&recipeId); err != nil {
		// Check if it's an item not found error
		missingItem := regItemNotFound.FindStringSubmatch(err.Error())
		if missingItem != nil {
			return ErrItemNotFound{Item: missingItem[1]}
		} else {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (me *recipeDbImpl) GetMissingRecipes(ctx context.Context, srcA string, listSrcB []string) (missing []string, err error) {
	var known []string
	row := me.c.QueryRow(ctx, `select v1_check_known_recipes($1, $2::text[])`, srcA, listSrcB)
	if err = row.Scan(&known); err != nil {
		return
	}
	missing = lo.Without(listSrcB, known...)
	return
}
