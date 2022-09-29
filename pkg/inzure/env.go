package inzure

import (
	"context"
	"fmt"
	"os"
)

var (
	// EnvSubscriptionJSON defines an environmental variable that can hold
	// a single filename referring to an inzure JSON file
	EnvSubscriptionJSON = "INZURE_JSON_FILE"
	// EnvSubscription defines an environmental variable that can hold a single
	// inzure subscription UUID and optional alias. To specify an alias, use an
	// = after the UUID.
	EnvSubscription = "INZURE_SUBSCRIPTION"
	// EnvSubscriptionFile defines an environmental variable that can hold a
	// file containing a newline separated list of subscription UUIDs and
	// optional aliases.
	EnvSubscriptionFile = "INZURE_SUBSCRIPTION_FILE"

	// EnvSubscriptionBatchFiles contains a list of files that should be used
	// when multiple subscriptions are possible. You can use the associated
	// BatchSubscriptionsFromEnv or BatchSubscriptionsFromEnvChan functions
	// to get these subscirptions.
	EnvSubscriptionBatchFiles = "INZURE_SUBSCRIPTION_BATCH_FILES"
)

func BatchFilesFromEnv() ([]string, error) {
	fname := os.Getenv(EnvSubscriptionBatchFiles)
	if fname == "" {
		return nil, fmt.Errorf(
			"%s environmental variable is not set",
			EnvSubscriptionBatchFiles,
		)
	}
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0)
	scanner := NewLineCommentScanner(f)
	for scanner.Scan() {
		fileName := scanner.Text()
		if fileName != "" {
			files = append(files, scanner.Text())
		}
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return files, nil
}

// SubscriptionIDsFromEnv will read the EnvSubscriptionFile and return a slice
// of SubscriptionIDs
func SubscriptionIDsFromEnv() ([]SubscriptionID, error) {
	fname := os.Getenv(EnvSubscriptionFile)
	if fname == "" {
		return nil, fmt.Errorf(
			"%s environmental variable is not set",
			EnvSubscriptionFile,
		)
	}
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	ids := make([]SubscriptionID, 0)
	scanner := NewLineCommentScanner(f)
	for scanner.Scan() {
		idString := scanner.Text()
		if idString != "" {
			ids = append(ids, SubIDFromString(scanner.Text()))
		}
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return ids, nil
}

// BatchSubscriptionsFromEnv will use the EnvSubscriptionBatchFiles
// environmental variable to load subscriptions into a slice. The passed
// password can be null if the files are unencrypted or the password can be
// pulled from the environment.
func BatchSubscriptionsFromEnv(pw []byte) ([]*Subscription, error) {
	names, err := BatchFilesFromEnv()
	if err != nil {
		return nil, err
	}
	subs := make([]*Subscription, 0, len(names))
	for _, fname := range names {
		sub, err := SubscriptionFromFilePassword(fname, pw)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// BatchSubscriptionsFromEnvChan will pull all subscriptions from the files in
// the EnvSubscriptionBatchFiles environmental variable. Errors are passed on
// the provided channel and the passed context can be used to stop everything.
// The provided password is only needed if the files are encrypted and the
// password can't be pulled out of the environment, otherwise it can be nil.
func BatchSubscriptionsFromEnvChan(
	ctx context.Context, pw []byte, ec chan<- error,
) <-chan *Subscription {
	c := make(chan *Subscription)
	go func() {
		defer close(c)
		done := ctx.Done()
		names, err := BatchFilesFromEnv()
		if err != nil {
			select {
			case ec <- err:
			case <-done:
				return
			}
		}
		for _, fname := range names {
			select {
			case <-done:
			default:
			}
			sub, err := SubscriptionFromFilePassword(fname, pw)
			if err != nil {
				select {
				case ec <- err:
				case <-done:
					return
				}
			}
			select {
			case c <- sub:
			case <-done:
				return
			}
		}
	}()
	return c
}
