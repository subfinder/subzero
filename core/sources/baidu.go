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

// Baidu is a source to process subdomains from https://baidu.com
type Baidu struct {
	lock *semaphore.Weighted
}

// ProcessDomain takes a given base domain and attempts to enumerate subdomains.
func (source *Baidu) ProcessDomain(ctx context.Context, domain string) <-chan *core.Result {
	if source.lock == nil {
		source.lock = defaultLockValue()
	}

	results := make(chan *core.Result)

	go func(domain string, results chan *core.Result) {
		defer close(results)

		if err := source.lock.Acquire(ctx, 1); err != nil {
			sendResultWithContext(ctx, results, core.NewResult(baiduLabel, nil, err))
			return
		}
		defer source.lock.Release(1)

		domainExtractor := core.NewSingleSubdomainExtractor(domain)

		for currentPage := 1; currentPage <= 750; currentPage++ {
			url := "https://www.baidu.com/s?rn=10&pn=" + strconv.Itoa(currentPage) + "&wd=site%3A" + domain + "+-www.+&oq=site%3A" + domain + "+-www.+"
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				sendResultWithContext(ctx, results, core.NewResult(baiduLabel, nil, err))
				return
			}

			req.Cancel = ctx.Done()
			req.WithContext(ctx)

			resp, err := core.HTTPClient.Do(req)
			if err != nil {
				sendResultWithContext(ctx, results, core.NewResult(baiduLabel, nil, err))
				return
			}

			if resp.StatusCode != 200 {
				resp.Body.Close()
				sendResultWithContext(ctx, results, core.NewResult(baiduLabel, nil, errors.New(resp.Status)))
				return
			}

			minLineLen := len(domain) + 2

			scanner := bufio.NewScanner(resp.Body)

			//scanner.Split(bufio.ScanWords)

			for scanner.Scan() {
				if ctx.Err() != nil {
					return
				}

				if len(scanner.Bytes()) < minLineLen {
					continue
				}

				str := domainExtractor(scanner.Bytes())

				if str != "" {
					//fmt.Println(scanner.Text())
					if !sendResultWithContext(ctx, results, core.NewResult(baiduLabel, str, nil)) {
						resp.Body.Close()
						return
					}
				}
			}

			resp.Body.Close()

			err = scanner.Err()

			if err != nil {
				sendResultWithContext(ctx, results, core.NewResult(baiduLabel, nil, err))
				return
			}
		}

	}(domain, results)
	return results
}
