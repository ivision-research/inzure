package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/Azure/go-autorest/autorest"
	"github.com/CarveSystems/inzure/pkg/inzure"
	"github.com/urfave/cli"
	"golang.org/x/net/proxy"
	"golang.org/x/term"
)

type GatherSocks5PoxyInfo struct {
	Address  string
	Username string
	Password string
}

var (
	GatherCertPath         string
	GatherSubscriptions    []inzure.SubscriptionID
	GatherSubscriptionFile string
	GatherTargets          string
	ExcludeGatherTargets   string
	GatherReportDir        string
	GatherVerbose          = false
	GatherSocks5Proxy      GatherSocks5PoxyInfo
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
	cli.StringFlag{
		Name:        "socks5-proxy",
		Usage:       "Host for a socks5 proxy. Format is host:[port]. Port defaults to 1080 if not specified",
		Destination: &GatherSocks5Proxy.Address,
	},
	cli.StringFlag{
		Name:        "socks5-user",
		Usage:       "Username for socks5 proxy if required",
		Destination: &GatherSocks5Proxy.Username,
	},
	cli.StringFlag{
		Name:        "socks5-pass",
		Usage:       "Password for socks5 proxy. If socks5-user is set and this is not, you will be prompted",
		Destination: &GatherSocks5Proxy.Password,
	},
	OutputFileFlag,
}

func scanGetSubscriptionsFromFile() error {
	if GatherSubscriptionFile == "" {
		return errors.New("must set either --sub or --sub-file")
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

func promptSocks5Password() string {
	pw, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		exitError(1, "failed to read password")
	}
	return string(pw)
}

func checkSocks5Params() {
	useSocks := len(GatherSocks5Proxy.Address) > 0
	hasUser := len(GatherSocks5Proxy.Username) > 0
	hasPassword := len(GatherSocks5Proxy.Password) > 0
	if !useSocks {
		if hasUser {
			exitError(1, "-socks5-user set without -socks5-proxy")
		} else if hasPassword {
			exitError(1, "-socks5-pass set without -socks5-proxy")
		}
		return
	}
	if hasUser {
		if !hasPassword {
			GatherSocks5Proxy.Password = promptSocks5Password()
		}
	}
}

func getProxy() proxy.Dialer {
	if len(GatherSocks5Proxy.Address) == 0 {
		return nil
	}
	var auth *proxy.Auth
	if len(GatherSocks5Proxy.Username) > 0 {
		auth = &proxy.Auth{
			User:     GatherSocks5Proxy.Username,
			Password: GatherSocks5Proxy.Password,
		}
	}
	pxy, err := proxy.SOCKS5("tcp", GatherSocks5Proxy.Address, auth, proxy.Direct)
	if err != nil {
		exitError(1, "failed to set up socks proxy")
	}
	return pxy
}

func CmdGather(c *cli.Context) {
	if GatherTargets == "list-all" {
		scanDumpTargetsList(os.Stdout)
		os.Exit(0)
	}

	checkSocks5Params()

	GatherSubscriptions = inzure.SubIDsFromStrings(c.StringSlice("sub"))
	if GatherSubscriptions == nil || len(GatherSubscriptions) == 0 {
		if err := scanGetSubscriptionsFromFile(); err != nil {
			exitError(1, err.Error())
		}
	}

	pw := maybeGetEncryptionPassword(c, false)

	if OutputFile == "" && len(GatherSubscriptions) == 1 {
		envFile := os.Getenv(inzure.EnvSubscriptionJSON)
		if envFile != "" {
			OutputFile = envFile
		}
	}

	pxy := getProxy()

	var wg sync.WaitGroup
	for _, id := range GatherSubscriptions {
		wg.Add(1)
		go func(subID inzure.SubscriptionID) {
			defer wg.Done()

			sub := inzure.NewSubscriptionFromID(subID)
			if pxy != nil {
				sub.SetProxy(pxy)
			}
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
				useExclude := ExcludeGatherTargets != ""
				for _, v := range inzure.AvailableTargets {
					sub.AddTarget(v)
				}
				if useExclude {
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
					if pxy != nil && isSocksConnectError(e) {
						cancel()
						fmt.Fprintln(os.Stderr, "failed to connect to socks proxy")
						break
					}
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

func isSocksConnectError(err error) bool {
	if err == nil {
		return false
	}
	for {

		if opErr, is := err.(*net.OpError); is {
			return opErr.Op == "socks connect"
		}

		cause := errors.Unwrap(err)
		if cause == nil {
			if oerr, is := err.(autorest.DetailedError); is {
				err = oerr.Original
			} else {
				return false
			}
		} else {
			err = cause
		}

	}
}
