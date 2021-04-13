package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/CarveSystems/inzure/pkg/inzure"
	"github.com/urfave/cli"
)

var CmdWhitelistFlags = []cli.Flag{
	InputFileFlag,
	OutputFileFlag,
}

type testFormat struct {
	Ignore    []string            `json:"ignore"`
	Whitelist map[string][]string `json:"whitelist"`
}

func CmdWhitelist(c *cli.Context) {
	if InputFile == "" {
		exitError(1, "need to specify an input inzure JSON with -f")
	}
	sub, err := inzure.SubscriptionFromFile(InputFile)
	if err != nil {
		exitError(1, err.Error())
	}
	tf := testFormat{
		Ignore:    make([]string, 0),
		Whitelist: make(map[string][]string),
	}
	nsgWhitelist(sub, tf.Whitelist)
	sqlWhitelist(sub, tf.Whitelist)
	cosmosWhitelist(sub, tf.Whitelist)
	postgresWhitelist(sub, tf.Whitelist)
	redisWhitelist(sub, tf.Whitelist)
	json.NewEncoder(os.Stdout).Encode(&tf)
}

func nsgWhitelist(sub *inzure.Subscription, whitelist map[string][]string) {
	into := make([]*inzure.NetworkSecurityGroup, 0)
	err := sub.FromQueryString("/NetworkSecurityGroups", &into)
	if err != nil {
		exitError(1, fmt.Sprintf("failed to get NetworkSecurityGroups: %v", err))
	}
	var key string
	for _, nsg := range into {
		qs := getIQS(nsg)
		for _, rule := range nsg.InboundRules {
			if !rule.Allows {
				continue
			}
			var portS string
			if len(rule.DestPorts) > 1 {
				ports := make([]string, len(rule.DestPorts))
				for i, port := range rule.DestPorts {
					ports[i] = port.String()
				}

				portS = strings.Join(ports, ",")
			} else if len(rule.DestPorts) == 1 {
				portS = rule.DestPorts[0].String()
			}
			if portS != "*" {
				key = fmt.Sprintf("%s:%s", qs, portS)
			} else {
				key = qs
			}
			if _, has := whitelist[key]; !has {
				whitelist[key] = make([]string, 0, 1)
			}
			for _, ip := range rule.SourceIPs {
				whitelist[key] = append(whitelist[key], ip.String())
			}
		}
	}
}

func genericFirewallWhitelist(sub *inzure.Subscription, iqs string, whitelist map[string][]string) {
	v, err := sub.ReflectFromQueryString(iqs)
	if err != nil {
		exitError(1, fmt.Sprintf("failed to get %s: %v", iqs, err))
	}
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	l := v.Len()
	for i := 0; i < l; i++ {
		e := v.Index(i)
		for e.Kind() == reflect.Ptr {
			e = e.Elem()
		}
		key := getIQS(e.Interface())
		fwall := e.FieldByName("Firewall")
		var rules []inzure.FirewallRule
		fwallRules, ok := fwall.Interface().(inzure.FirewallRules)
		if !ok {
			exitError(1, fmt.Sprintf("%v is not valid for a generic firewall: %v", e.Type(), fwall.Type()))
		} else {
			rules = []inzure.FirewallRule(fwallRules)
		}
		if _, ok := whitelist[key]; !ok {
			whitelist[key] = make([]string, 0, 1)
		}
		for _, rule := range rules {
			whitelist[key] = append(whitelist[key], rule.IPRange.String())
		}
	}
}

func cosmosWhitelist(sub *inzure.Subscription, whitelist map[string][]string) {
}

func postgresWhitelist(sub *inzure.Subscription, whitelist map[string][]string) {
	genericFirewallWhitelist(sub, "/PostgresServers", whitelist)
}

func sqlWhitelist(sub *inzure.Subscription, whitelist map[string][]string) {
	genericFirewallWhitelist(sub, "/SQLServers", whitelist)
}

func redisWhitelist(sub *inzure.Subscription, whitelist map[string][]string) {
	into := make([]*inzure.RedisServer, 0)
	err := sub.FromQueryString("/RedisServers", &into)
	if err != nil {
		exitError(1, fmt.Sprintf("failed to get /RedisServers: %v", err))
	}
	for _, serv := range into {
		key := getIQS(serv)
		if _, has := whitelist[key]; !has {
			whitelist[key] = make([]string, 0, 1)
		}
		if len(serv.Firewall) == 0 {
			whitelist[key] = append(whitelist[key], "*")
		} else {
			for _, rule := range serv.Firewall {
				whitelist[key] = append(whitelist[key], rule.IPRange.String())
			}
		}
	}
}

func getIQS(v interface{}) string {
	key, err := inzure.ToQueryString(v)
	if err != nil {
		exitError(1, fmt.Sprintf("failed to get IQS for %v: %v", v, err))
	}
	return key
}
