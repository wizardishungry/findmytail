package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/zellyn/kooky"
	_ "github.com/zellyn/kooky/browser/chrome"
)

const findMyURL = `https://www.icloud.com/find`

func main() {
	flag.Parse()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.CombinedOutput(os.Stderr),
		// Run headed for now; still getting a ton of login prompts.
		chromedp.Flag("headless", false),
		chromedp.WindowSize(1024, 768),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// also set up a custom logger
	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// ensure that the browser process is started
	if err := chromedp.Run(taskCtx); err != nil {
		log.Fatal(err)
	}

	if err := chromedp.Run(taskCtx,
		copyCookies(),
		intercept(),
		chromedp.Navigate(findMyURL),
		dumpBodies(),
	); err != nil {
		log.Fatal(err)
	}
}

func intercept() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		err := fetch.Enable().WithPatterns([]*fetch.RequestPattern{
			{
				URLPattern:   `*/refreshClient*`,
				RequestStage: fetch.RequestStageResponse,
			},
		}).Do(ctx)
		if err != nil {
			return fmt.Errorf("fetch.Enable: %w", err)
		}

		return nil
	})
}

func dumpBodies() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		c := make(chan *fetch.EventRequestPaused, 1)
		go chromedp.ListenTarget(ctx, func(ev interface{}) {
			switch ev := ev.(type) {
			case *fetch.EventRequestPaused:
				c <- ev
			}
		})

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		for ev := range c {
			if ev.ResponseStatusCode != http.StatusOK {
				log.Println("ResponseStatusCode unexpected ", ev.ResponseStatusCode)
				continue
			}

			body, err := fetch.GetResponseBody(ev.RequestID).Do(ctx)
			if err != nil {
				log.Println("fetch.GetResponseBody: ", err)
				continue
			}

			if err := fetch.ContinueRequest(ev.RequestID).Do(ctx); err != nil {
				log.Println("fetch.ContinueRequest: ", err)
				continue
			}

			var js json.RawMessage
			if err := json.Unmarshal(body, &js); err != nil {
				log.Println("json.Unmarshal", err)
				continue
			}
			if err := enc.Encode(js); err != nil {
				log.Println("json.Encode", err)
				continue
			}
		}

		return nil
	})
}

func copyCookies() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {

		cookies := kooky.ReadCookies(
			kooky.FilterFunc(func(cookie *kooky.Cookie) bool {
				return cookie != nil && (strings.HasSuffix(cookie.Domain, `apple.com`) ||
					strings.HasSuffix(cookie.Domain, `icloud.com`))
			}),
		)

		for _, cookie := range cookies {
			sc := network.SetCookie(cookie.Name, cookie.Value).
				WithPath(cookie.Path).
				WithExpires((*cdp.TimeSinceEpoch)(&cookie.Expires)).
				WithDomain(cookie.Domain).
				WithSecure(cookie.Secure).
				WithHTTPOnly(cookie.HttpOnly)

			{
				var sameSite network.CookieSameSite
				switch cookie.SameSite {
				default:
					fallthrough
				case http.SameSiteDefaultMode:
					sameSite = ""
				case http.SameSiteLaxMode:
					sameSite = network.CookieSameSiteLax
				case http.SameSiteStrictMode:
					sameSite = network.CookieSameSiteStrict
				case http.SameSiteNoneMode:
					sameSite = network.CookieSameSiteNone
				}
				if sameSite != "" {
					sc.WithSameSite(sameSite)
				}
			}
			err := sc.Do(ctx)
			if err != nil {
				return fmt.Errorf("SetCookie: %w", err)
			}
		}

		return nil

	})
}
