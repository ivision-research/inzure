package inzure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
)

// VirtualMachine holds the data for a given Virtual Machine. note that this
// type is intended to collect information about both new and classical VMs.
type VirtualMachine struct {
	Meta                    ResourceID
	ComputerName            string
	IsClassic               bool
	AdminUser               string
	DisablePasswordAuth     UnknownBool
	SSHKeys                 []SSHPublicKey
	AutomaticUpdates        UnknownBool
	WindowsRMListeners      []WindowsRMListener
	NetworkInterfaces       []NetworkInterface
	PrimaryNetworkInterface ResourceID
	OsName                  string
	OsVersion               string
	CustomData              string
	OsType                  OsType
	Disks                   []VMDisk
}

type OsType uint8

const (
	OsTypeUnknown OsType = iota
	OsTypeLinux
	OsTypeWindows
)

// DiskEncryption holds the location of an encryption key and whether that key
// is enabled for the given disk
type DiskEncryption struct {
	Enabled          UnknownBool
	EncryptionKey    string
	KeyEncryptionKey string
}

// VMDisk contains the name and encryption information for the disk
type VMDisk struct {
	Name               string
	EncryptionSettings []DiskEncryption
}

func NewEmptyVMDisk() VMDisk {
	vmd := VMDisk{
		EncryptionSettings: make([]DiskEncryption, 0),
	}
	return vmd
}

func NewEmptyVirtualMachine() *VirtualMachine {
	vm := &VirtualMachine{
		SSHKeys:            make([]SSHPublicKey, 0),
		WindowsRMListeners: make([]WindowsRMListener, 0),
		Disks:              make([]VMDisk, 0),
	}
	vm.Meta.setupEmpty()
	vm.PrimaryNetworkInterface.setupEmpty()
	return vm
}

// WindowsRMListener is a listener for Windows VMs.
type WindowsRMListener struct {
	IsHTTPS        UnknownBool
	CertificateURL string
}

// SSHPublicKey contains the key itself as a string and the location on the VM
type SSHPublicKey struct {
	Path      string
	PublicKey string
}

func (vm *VirtualMachine) FromAzure(az *armcompute.VirtualMachine) {
	vm.Meta.setupEmpty()
	if az.ID != nil {
		vm.Meta.fromID(*az.ID)
	}
	props := az.Properties
	if props == nil {
		return
	}

	if props.OSProfile != nil {
		vm.loadOSProfile(props.OSProfile)
	}

	if props.NetworkProfile != nil {
		vm.loadNetworkProfile(props.NetworkProfile)
	}

	if props.InstanceView != nil {
		vm.loadInstanceView(props.InstanceView)
	}
}
func (vm *VirtualMachine) loadInstanceView(iv *armcompute.VirtualMachineInstanceView) {
	gValFromPtr(&vm.OsName, iv.OSName)
	gValFromPtr(&vm.OsVersion, iv.OSVersion)
	if iv.Disks != nil {
		for _, d := range iv.Disks {
			if d.Name != nil {
				disk := NewEmptyVMDisk()
				disk.Name = *d.Name
				if d.EncryptionSettings != nil {
					for _, s := range d.EncryptionSettings {
						var setting DiskEncryption
						setting.Enabled.FromBoolPtr(s.Enabled)
						if s.DiskEncryptionKey != nil {
							gValFromPtr(&setting.EncryptionKey, s.DiskEncryptionKey.SecretURL)
						}
						if s.KeyEncryptionKey != nil {
							gValFromPtr(&setting.KeyEncryptionKey, s.KeyEncryptionKey.KeyURL)
						}
						disk.EncryptionSettings = append(disk.EncryptionSettings, setting)
					}
				}
				vm.Disks = append(vm.Disks, disk)
			}
		}
	}

}

func (vm *VirtualMachine) loadNetworkProfile(netp *armcompute.NetworkProfile) {
	if netp.NetworkInterfaces != nil {
		for _, aziface := range netp.NetworkInterfaces {
			if aziface.ID != nil {
				var ni NetworkInterface
				ni.setupEmpty()
				ni.Meta.fromID(*aziface.ID)
				vm.NetworkInterfaces = append(vm.NetworkInterfaces, ni)
				props := aziface.Properties
				if props != nil &&
					props.Primary != nil {
					if *props.Primary {
						vm.PrimaryNetworkInterface = ni.Meta
					}
				}
			}
		}
	}
}

func (vm *VirtualMachine) loadOSProfile(os *armcompute.OSProfile) {
	gValFromPtr(&vm.ComputerName, os.ComputerName)
	gValFromPtr(&vm.AdminUser, os.AdminUsername)
	gValFromPtr(&vm.CustomData, os.CustomData)

	if os.WindowsConfiguration != nil {

		vm.OsType = OsTypeWindows

		win := os.WindowsConfiguration
		// TODO: I think this is always false..?
		vm.DisablePasswordAuth = BoolFalse
		if win.EnableAutomaticUpdates != nil {
			vm.AutomaticUpdates = unknownFromBool(*win.EnableAutomaticUpdates)
		}
		if win.WinRM != nil && win.WinRM.Listeners != nil {
			for _, rml := range win.WinRM.Listeners {
				var url string
				if rml.CertificateURL != nil {
					url = *rml.CertificateURL
				}
				isHttps := BoolUnknown
				if rml.Protocol != nil {
					isHttps = unknownFromBool(*rml.Protocol == armcompute.ProtocolTypesHTTPS)
				}
				vm.WindowsRMListeners = append(
					vm.WindowsRMListeners,
					WindowsRMListener{
						IsHTTPS:        isHttps,
						CertificateURL: url,
					},
				)
			}
		}
	} else if os.LinuxConfiguration != nil {

		vm.OsType = OsTypeLinux

		lin := os.LinuxConfiguration
		vm.DisablePasswordAuth.FromBoolPtr(lin.DisablePasswordAuthentication)
		if lin.SSH != nil && lin.SSH.PublicKeys != nil {
			for _, pk := range lin.SSH.PublicKeys {
				var pubKey SSHPublicKey
				if pk.Path != nil {
					pubKey.Path = *pk.Path
				}
				if pk.KeyData != nil {
					pubKey.PublicKey = *pk.KeyData
				}
				vm.SSHKeys = append(vm.SSHKeys, pubKey)
			}
		}
	}
}
