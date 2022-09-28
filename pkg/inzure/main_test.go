package inzure

import (
	"os"
	"testing"
	"time"
)

const subId = "d707bd34-56bd-4ff3-ba5d-fd3b1c3d71d6"
const rgAName = "rgA"
const rgBName = "rgB"

func newResourceId(rg string, tag AzureResourceTag, name string) ResourceID {
	return ResourceID{
		RawID:             "",
		Subscription:      subId,
		ResourceGroupName: rg,
		Tag:               tag,
		Name:              name,
	}
}

var auditDate = time.Now()

var testSub Subscription = Subscription{
	ID:                     subId,
	Alias:                  "Test Subscription",
	AuditDate:              auditDate,
	ResourceGroups:         map[string]*ResourceGroup{},
	ClassicStorageAccounts: []*StorageAccount{},
}

func TestMain(m *testing.M) {
	var rgA = NewEmptyResourceGroup()
	var rgB = NewEmptyResourceGroup()

	nsgAA := NewEmptyNSG()
	nsgAA.Meta = newResourceId(rgAName, NetworkSecurityGroupT, "nsgAA")
	nsgAA.InboundRules = append(nsgAA.InboundRules, SecurityRule{
		Name:        "nsgAA",
		Allows:      true,
		Inbound:     true,
		Priority:    10,
		Description: "",
		SourceIPs:   createIPs("12.34.56.78"),
		SourcePorts: createPorts("*"),
		DestIPs:     createIPs("*"),
		DestPorts:   createPorts("*"),
	})
	rgA.NetworkSecurityGroups = append(rgA.NetworkSecurityGroups, nsgAA)
	testSub.ResourceGroups[rgAName] = rgA
	testSub.ResourceGroups[rgBName] = rgB
	os.Exit(m.Run())
}
