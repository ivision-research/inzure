package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"

	"github.com/CarveSystems/inzure"
	"github.com/codegangsta/cli"
)

var (
	GatherCertPath         string
	GatherSubscriptions    []inzure.SubscriptionID
	GatherSubscriptionFile string
	GatherTargets          string
	ExcludeGatherTargets   string
	GatherReportDir        string
	GatherVerbose          = false
	GatherNoListKeys       = false
)

var CmdGatherFlags = []cli.Flag{
	cli.StringSliceFlag{
		Name:   "sub",
		EnvVar: inzure.EnvSubscription,
		Usage:  "Subscription UUID to scan (optional alias specified by an = and alias after the UUID). This flag can be specified more than once or you can use --sub-file instead.",
	},
	cli.StringFlag{
		Name:        "sub-file",
		EnvVar:      inzure.EnvSubscriptionFile,
		Usage:       "A file containing subscriptions to scan. Each subscription should be on its own line. Lines with a preceding # are ignored.",
		Destination: &GatherSubscriptionFile,
	},
	cli.StringFlag{
		Name:        "targets",
		Usage:       "A comma separated list of targets. Set to \"list-all\" to view all options. If not set all targets are set",
		Destination: &GatherTargets,
	},
	cli.StringFlag{
		Name:        "exclude",
		Usage:       "A comma separated list of targets to exclude. This can't be set at the same time as \"targets\".",
		Destination: &ExcludeGatherTargets,
	},

	cli.BoolFlag{
		Name:        "no-storage-keys",
		Usage:       "Don't try to use storage account keys. This will also disable container searching.",
		Destination: &GatherNoListKeys,
	},
	cli.BoolFlag{
		Name:        "v",
		Usage:       "Verbose output",
		Destination: &GatherVerbose,
	},
	cli.StringFlag{
		Name:        "cert",
		Usage:       "Enable classic resource support by providing a certificate",
		Destination: &GatherCertPath,
	},
	cli.StringFlag{
		Name:        "dir",
		Usage:       "Directory to output reports to",
		Destination: &GatherReportDir,
	},
	OutputFileFlag,
}

func scanGetSubscriptionsFromFile() error {
	if GatherSubscriptionFile == "" {
		return errors.New("no scan-file set")
	}
	f, err := os.Open(GatherSubscriptionFile)
	if err != nil {
		return err
	}
	scan := inzure.NewLineCommentScanner(f)
	for scan.Scan() {
		line := scan.Text()
		if GatherVerbose {
			fmt.Println("Searching subscription:", line)
		}
		GatherSubscriptions = append(
			GatherSubscriptions, inzure.SubIDFromString(line),
		)
	}
	return nil
}

func scanDumpTargetsList(w io.Writer) {
	for k := range inzure.AvailableTargets {
		_, err := w.Write([]byte(fmt.Sprintf("- %s\n", k)))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func CmdGather(c *cli.Context) {
	if GatherTargets == "list-all" {
		scanDumpTargetsList(os.Stdout)
		os.Exit(0)
	}

	GatherSubscriptions = inzure.SubIDsFromStrings(c.StringSlice("sub"))
	if GatherSubscriptions == nil || len(GatherSubscriptions) == 0 {
		if err := scanGetSubscriptionsFromFile(); err != nil {
			exitError(1, err.Error())
		}
	}

	pw := getEncryptPassword(c)

	if OutputFile == "" && len(GatherSubscriptions) == 1 {
		envFile := os.Getenv(inzure.EnvSubscriptionJSON)
		if envFile != "" {
			OutputFile = envFile
		}
	}

	var wg sync.WaitGroup
	for _, id := range GatherSubscriptions {
		wg.Add(1)
		go func(subID inzure.SubscriptionID) {
			defer wg.Done()

			sub := inzure.NewSubscriptionFromID(subID)
			sub.HasListKeysPermission(!GatherNoListKeys)
			sub.SetQuiet(!GatherVerbose)
			if GatherTargets != "" {
				if ExcludeGatherTargets != "" {
					exitError(1, "both --targets and --exclude can't be set")
				}
				spl := strings.Split(GatherTargets, ",")
				for _, s := range spl {
					v, ok := inzure.AvailableTargets[s]
					if !ok {
						exitError(1, "unknown target %s", s)
					}
					sub.AddTarget(v)
				}
			} else {
				for _, v := range inzure.AvailableTargets {
					sub.AddTarget(v)
				}
				if ExcludeGatherTargets != "" {
					spl := strings.Split(ExcludeGatherTargets, ",")
					for _, s := range spl {
						v, ok := inzure.AvailableTargets[s]
						if !ok {
							exitError(1, "unknown target %s", s)
						}
						sub.UnsetTarget(v)
					}
				}
			}
			if GatherCertPath != "" {
				f, err := os.Open(GatherCertPath)
				if err != nil {
					exitError(
						1, "failed to open classic key file %s: %v",
						GatherCertPath, err,
					)
				}
				key := make([]byte, 0, 1024)
				tmp := make([]byte, 256)
				for {
					n, err := f.Read(tmp)
					if err != nil {
						if err == io.EOF {
							break
						} else {
							exitError(
								1, "error reading classic key file %s: %v",
								GatherCertPath, err,
							)
						}
					}
					key = append(key, tmp[:n]...)
				}
				sub.SetClassicKey(key)
			}
			ec := make(chan error, 10)
			ctx, cancel := context.WithCancel(context.Background())

			doneChan := make(chan struct{}, 1)
			go func() {
				c := make(chan os.Signal, 1)
				signal.Notify(c, os.Interrupt)
				defer func() {
					cancel()
					signal.Reset(os.Interrupt)
					close(c)
				}()
				select {
				case <-doneChan:
				case <-c:
					fmt.Fprintf(os.Stderr, "Ending search early due to interrupt")
				}
			}()
			go func() {
				for e := range ec {
					fmt.Fprintln(os.Stderr, e)
				}
				doneChan <- struct{}{}
			}()
			var fname string
			sub.SearchAllTargets(ctx, ec)
			if OutputFile == "" {
				tString := sub.AuditDate.Format("02-01-2006-15:04")
				identifier := strings.Replace(
					strings.Replace(sub.Alias, " ", "-", -1),
					string(os.PathSeparator), "_", -1,
				)
				if identifier == "" {
					identifier = sub.ID
				}
				fname = path.Join(
					GatherReportDir,
					fmt.Sprintf("%s-inzure-%s.json", tString, identifier),
				)
			} else {
				fname = OutputFile
			}
			if pw != nil {
				fname += inzure.EncryptedFileExtension
			}
			f, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
			if err != nil {
				errorf("failed to open output file: %v", err)
				return
			}
			if pw != nil {
				err := inzure.EncryptSubscriptionAsJSON(&sub, pw, f)
				if err != nil {
					exitError(1, "failed to write encrypted JSON: %v", err)
				}
			} else {
				if GatherVerbose {
					fmt.Fprintln(
						os.Stderr,
						"[WARNING] No encryption password was set!",
					)
				}
				if err := json.NewEncoder(f).Encode(&sub); err != nil {
					errorf("error outputing JSON: %v", err)
				}
			}
		}(id)
	}
	wg.Wait()
}
