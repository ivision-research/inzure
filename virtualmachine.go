package inzure

import "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"

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
	IsHTTPS        bool
	CertificateURL string
}

// SSHPublicKey contains the key itself as a string and the location on the VM
type SSHPublicKey struct {
	Path      string
	PublicKey string
}

func (v *VirtualMachine) FromAzure(az *compute.VirtualMachine) {
	v.Meta.setupEmpty()
	if az.ID != nil {
		v.Meta.fromID(*az.ID)
	}
	props := az.VirtualMachineProperties
	if props == nil {
		return
	}
	if props.OsProfile != nil {
		os := props.OsProfile
		if os.ComputerName != nil {
			v.ComputerName = *os.ComputerName
		}
		if os.AdminUsername != nil {
			v.AdminUser = *os.AdminUsername
		}
		if os.WindowsConfiguration != nil {

			v.OsType = OsTypeWindows

			win := os.WindowsConfiguration
			// TODO: I think this is always false..?
			v.DisablePasswordAuth = BoolFalse
			if win.EnableAutomaticUpdates != nil {
				v.AutomaticUpdates = unknownFromBool(*win.EnableAutomaticUpdates)
			}
			if win.WinRM != nil && win.WinRM.Listeners != nil {
				for _, rml := range *win.WinRM.Listeners {
					var url string
					if rml.CertificateURL != nil {
						url = *rml.CertificateURL
					}
					v.WindowsRMListeners = append(
						v.WindowsRMListeners,
						WindowsRMListener{
							IsHTTPS:        rml.Protocol == compute.HTTPS,
							CertificateURL: url,
						},
					)
				}
			}
		} else if os.LinuxConfiguration != nil {

			v.OsType = OsTypeLinux

			lin := os.LinuxConfiguration
			if lin.DisablePasswordAuthentication != nil {
				v.DisablePasswordAuth = unknownFromBool(*lin.DisablePasswordAuthentication)
			}
			if lin.SSH != nil && lin.SSH.PublicKeys != nil {
				for _, pk := range *lin.SSH.PublicKeys {
					var pubKey SSHPublicKey
					if pk.Path != nil {
						pubKey.Path = *pk.Path
					}
					if pk.KeyData != nil {
						pubKey.PublicKey = *pk.KeyData
					}
					v.SSHKeys = append(v.SSHKeys, pubKey)
				}
			}
		}
	}

	if props.NetworkProfile != nil {
		netp := props.NetworkProfile
		if netp.NetworkInterfaces != nil {
			for _, aziface := range *netp.NetworkInterfaces {
				if aziface.ID != nil {
					var ni NetworkInterface
					ni.setupEmpty()
					ni.Meta.fromID(*aziface.ID)
					v.NetworkInterfaces = append(v.NetworkInterfaces, ni)
					if aziface.NetworkInterfaceReferenceProperties != nil &&
						aziface.NetworkInterfaceReferenceProperties.Primary != nil {
						if *aziface.NetworkInterfaceReferenceProperties.Primary {
							v.PrimaryNetworkInterface = ni.Meta
						}
					}
				}
			}
		}
	}

	if props.InstanceView != nil {
		iv := props.InstanceView
		if iv.OsName != nil {
			v.OsName = *iv.OsName
		}
		if iv.OsVersion != nil {
			v.OsVersion = *iv.OsVersion
		}
		if iv.Disks != nil {
			for _, d := range *iv.Disks {
				if d.Name != nil {
					disk := NewEmptyVMDisk()
					disk.Name = *d.Name
					if d.EncryptionSettings != nil {
						for _, s := range *d.EncryptionSettings {
							var setting DiskEncryption
							if s.Enabled != nil {
								setting.Enabled = unknownFromBool(*s.Enabled)
							}
							if s.DiskEncryptionKey != nil && s.DiskEncryptionKey.SecretURL != nil {
								setting.EncryptionKey = *s.DiskEncryptionKey.SecretURL
							}
							if s.KeyEncryptionKey != nil && s.KeyEncryptionKey.KeyURL != nil {
								setting.KeyEncryptionKey = *s.KeyEncryptionKey.KeyURL
							}
							disk.EncryptionSettings = append(disk.EncryptionSettings, setting)
						}
					}
					v.Disks = append(v.Disks, disk)
				}
			}
		}
	}
}
