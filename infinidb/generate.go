package infinidb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"

	"binder.fun/infinicraft/infinidb/rdb"
	"github.com/samber/lo"
	"golang.org/x/time/rate"
)

var baseApiUrl = "https://neal.fun/api/infinite-craft/pair"

type ItemGenerator struct {
	// Database connection, only required field
	Db rdb.RecipeDb
	// HTTP client for querying the neal.fun server
	Client *http.Client
	// Rate limit for querying the server
	QueryLimiter *rate.Limiter
	// Items that are totally new
	Discoveries map[string]bool
	// Cancel context
	ctx context.Context
}

type recipePairs = lo.Tuple2[string, []string]

type infinicraftResponse struct {
	Result string `json:"result"`
	Emoji  string `json:"emoji"`
	IsNew  bool   `json:"isNew"`
}

func NewItemGenerator(db rdb.RecipeDb, ctx context.Context) *ItemGenerator {
	if db == nil {
		panic("nil db")
	}
	return &ItemGenerator{
		Db:           db,
		Client:       new(http.Client),
		QueryLimiter: rate.NewLimiter(rate.Limit(2.0), 1),
		Discoveries:  make(map[string]bool),
		ctx:          ctx,
	}
}

func (gen *ItemGenerator) Generate(initial []string) (output []Recipe, err error) {
	var (
		missing    []string
		newRecipes []Recipe
	)

	// Generate all possible recipe permutations from the initial input
	permutations := make(chan recipePairs)
	defer close(permutations)
	go gen.recipePermutations(initial, permutations)

	// Process the permutations
	output = make([]Recipe, 0)
	for {
		select {
		case <-gen.ctx.Done():
			return nil, gen.ctx.Err()
		case task := <-permutations:
			// Check DB first
			if missing, err = gen.Db.GetMissingRecipes(gen.ctx, task.A, task.B); err != nil {
				return
			}
			// Query for new recipes
			if newRecipes, err = gen.queryRecipes(task.A, missing); err != nil {
				return
			}
			output = append(output, newRecipes...)
		}
	}
}

func (gen *ItemGenerator) queryRecipes(srcA string, srcB []string) (output []Recipe, err error) {
	var (
		req    *http.Request
		res    *http.Response
		apiRes infinicraftResponse
	)
	output = make([]Recipe, 0, len(srcB))
	for _, bb := range srcB {
		// For each unknown recipe, call out to the api, rate-limited
		if err := gen.QueryLimiter.Wait(gen.ctx); err != nil {
			return nil, err
		}
		// Build query params
		q := make(url.Values)
		q.Add("first", srcA)
		q.Add("second", bb)
		if req, err = http.NewRequestWithContext(gen.ctx, "GET", baseApiUrl+"?"+q.Encode(), nil); err != nil {
			return
		}
		// Do the http request
		if res, err = gen.Client.Do(req); err != nil {
			return
		}
		defer res.Body.Close()
		// Parse and handle response
		if err = json.NewDecoder(res.Body).Decode(&apiRes); err != nil {
			return
		}
		output = append(output, Recipe{srcA, bb, apiRes.Result})
		if apiRes.IsNew {
			gen.Discoveries[apiRes.Result] = true
		}
	}
	return
}

func (gen *ItemGenerator) recipePermutations(inputs []string, output chan<- recipePairs) {
	slices.Sort(inputs)
	for i, srcA := range inputs {
		output <- recipePairs{A: srcA, B: inputs[i+1:]}
	}
}
