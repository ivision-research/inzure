module github.com/CarveSystems/inzure/cmd/inzure

go 1.12

replace github.com/CarveSystems/inzure/pkg/inzure => ../../pkg/inzure

require (
	github.com/Azure/go-autorest v11.7.0+incompatible // indirect
	github.com/CarveSystems/inzure/pkg/inzure v1.0.0
	github.com/chzyer/logex v1.1.10 // indirect
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/chzyer/test v0.0.0-20180213035817-a1ea475d72b1 // indirect
	github.com/urfave/cli v1.22.5
	golang.org/x/net v0.0.0-20220906165146-f3363e06e74c
	golang.org/x/sys v0.0.0-20220906165534-d0df966e6959 // indirect
	golang.org/x/term v0.0.0-20220722155259-a9ba230a4035
)
