package core

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type FakeSource1 struct{}
type FakeSource2 struct{}

func (s *FakeSource1) ProcessDomain(ctx context.Context, domain string) <-chan *Result {
	results := make(chan *Result)

	go func(domain string) {
		defer close(results)
		for _, subdomain := range []string{"a.", "b.", "c."} {
			time.Sleep(2 * time.Second)
			results <- &Result{Success: subdomain + domain}
		}
	}(domain)
	return results
}

func (s *FakeSource2) ProcessDomain(ctx context.Context, domain string) <-chan *Result {
	results := make(chan *Result)

	go func(domain string) {
		defer close(results)
		for _, subdomain := range []string{"admin.", "user.", "mod."} {
			time.Sleep(2 * time.Second)
			results <- &Result{Success: subdomain + domain}
		}
	}(domain)
	return results
}

func TestEnumerateSubdomains(t *testing.T) {
	domain := "google.com"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	options := &EnumerationOptions{
		Sources: []Source{&FakeSource1{}, &FakeSource2{}},
	}

	counter := 0

	for result := range EnumerateSubdomains(ctx, domain, options) {
		counter++
		fmt.Println(result)
	}

	if counter != 6 {
		t.Error(counter)
	}
}

func TestEnumerateSubdomains_Recursively(t *testing.T) {
	domain := "google.com"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	options := &EnumerationOptions{
		Sources:   []Source{&FakeSource1{}, &FakeSource2{}},
		Recursive: true,
	}

	counter := 0

	for result := range EnumerateSubdomains(ctx, domain, options) {
		counter++
		if counter == 15 {
			cancel()
		}
		fmt.Println(result)
	}

	t.Log(counter)
}

func TestEnumerateSubdomains_Recursively_UniqResults(t *testing.T) {
	domain := "google.com"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	options := &EnumerationOptions{
		Sources:   []Source{&FakeSource1{}, &FakeSource2{}},
		Recursive: true,
		Uniq:      true,
	}

	counter := 0

	results := EnumerateSubdomains(ctx, domain, options)

	for result := range UniqResults(results) {
		counter++
		fmt.Println(result)
	}

	fmt.Println(counter, ctx.Err())

}

func ExampleEnumerateSubdomains() {
	domain := "google.com"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sources := []Source{
		&FakeSource1{},
		&FakeSource2{},
	}

	options := &EnumerationOptions{
		Sources: sources,
	}

	counter := 0

	for result := range EnumerateSubdomains(ctx, domain, options) {
		if result.Failure == nil {
			counter++
		}
	}

	fmt.Println(counter)
	// Output: 6
}
