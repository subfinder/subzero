package sources

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/subfinder/research/core"
	"golang.org/x/sync/semaphore"
)

// DogPile is a source to process subdomains from http://dogpile.com
//
// Note
//
// This source uses http instead of https because of problems dogpile's SSL cert.
//
type DogPile struct {
	lock *semaphore.Weighted
}

// ProcessDomain takes a given base domain and attempts to enumerate subdomains.
func (source *DogPile) ProcessDomain(ctx context.Context, domain string) <-chan *core.Result {
	if source.lock == nil {
		source.lock = defaultLockValue()
	}

	results := make(chan *core.Result)

	go func(domain string, results chan *core.Result) {
		defer close(results)

		if err := source.lock.Acquire(ctx, 1); err != nil {
			sendResultWithContext(ctx, results, core.NewResult(dogpileLabel, nil, err))
			return
		}
		defer source.lock.Release(1)

		domainExtractor := core.NewSingleSubdomainExtractor(domain)

		for currentPage := 1; currentPage <= 750; currentPage++ {
			url := "http://www.dogpile.com/search/web?q=" + domain + "&qsi=" + strconv.Itoa(currentPage*15+1)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				sendResultWithContext(ctx, results, core.NewResult(dogpileLabel, nil, err))
				return
			}

			req.Cancel = ctx.Done()
			req.WithContext(ctx)

			resp, err := core.HTTPClient.Do(req)
			if err != nil {
				sendResultWithContext(ctx, results, core.NewResult(dogpileLabel, nil, err))
				return
			}

			if resp.StatusCode != 200 {
				resp.Body.Close()
				sendResultWithContext(ctx, results, core.NewResult(dogpileLabel, nil, errors.New(resp.Status)))
				return
			}

			scanner := bufio.NewScanner(resp.Body)

			scanner.Split(bufio.ScanWords)

			for scanner.Scan() {
				if ctx.Err() != nil {
					resp.Body.Close()
					return
				}
				str := domainExtractor(scanner.Bytes())
				if str != "" {
					if !sendResultWithContext(ctx, results, core.NewResult(dogpileLabel, str, nil)) {
						resp.Body.Close()
						return
					}
				}
			}

			err = scanner.Err()

			if err != nil {
				resp.Body.Close()
				sendResultWithContext(ctx, results, core.NewResult(dogpileLabel, nil, err))
				return
			}

			resp.Body.Close()
		}

	}(domain, results)
	return results
}
