# Contributing

Anyone is welcome to contribute to this product to make it better. For the most 
part Go will enforce style guidelines on you, but the following is some helpful 
information.

Also any tests are appreciated!

## Steps for adding a new resource

If you'd like to add a new resource here are the recommended steps:

1. Write a wrapper struct conforming to the idea in the [Types](#types) 
   section.
2. Modify the `AzureAPI` interface to include a generator function for your 
   type as shown in the [Azure API](#azure-api) section.
3. Add a slice of your type to the `ResourceGroup` struct. Don't forget to
   modify the `newResourceGroup` function to initialize this slice. The slice
   should be named a pluralized version of the struct name.
5. Add a target value for your type over in the [the subscription 
   file](subscription.go). This includes adding a string identifier for your 
   target and adding it to the `AvailableTargets` map.
6. Add the search logic for your target into the `SearchAllTargets` method of 
   `Subscription`.

Extra work that could be done is adding a test over the `inzure-test` repo or 
writing one that can be used as a plugin.

## Types

This package is intended to abstract away any Azure dependencies and trim out 
unnecessary data. If you find yourself adding something that relies on piece of 
Azure data, wrap it in a custom type with a `FromAzure` method. You'll see the 
primary initialization method for these types is:

```.go
// NewEmptyFoo should return an instance that can be serialized immediately
// and not have any null values.
foo := NewEmptyFoo()
foo.FromAzure(&AzureFoo)
```

Try to maintain that style. Upon exiting a `FromAzure` function there is no 
guarantee any data that was in there previously is the same as it was before 
entering.

Most (all?) nonclassic types have a `ResourceID` which is derived from their ID
string. These _must_ be placed in your type as the field `Meta` if they are
applicable. If they are not, there should be a `Name` field in your struct that
is a string. This is essential for Query String functionality. These tags are
essential for cross referencing and need to be included with every Azure type
that has an ID string of the form
`/subscription/{sub}/tag/val/tag/val/tag/val/..`

## Azure API

The AzureAPI interface is exported along with a way to construct a default 
implementation, but most of the time this is not something a user will need.  
All of those functions make use the Go "generator" pattern with a passed 
context (except the channels are typically buffered):

```.go
c := make(chan Foo, buf)
go func() {
    defer close(c)
    // do work
    foo := Foo{}
    select {
      case <-ctx.Done():
        return
      case c <- foo:
    }
}()
return c
```

The intention here is to keep the interactions with the Azure API async. You
can spawn more goroutines in your method if necessary.

## Pseudo Enums

There are quite a few pseudo enums used in this code. By pseudo enums I mean
the Go idiom:

```.go
type Foo uint
const (
    FooBar Foo = iota
    FooBaz
    FooBarBaz
)
```

These types should implement the `Stringer` interface either automatically,
using a `//go:generate stringer -type TYPE` command, or manually implementing
the `func (f *foo) String() string` method.  Also, the 0 value of any of these
should indicate that it was unset.

The above would then actually look like:

```.go

//go:generate stringer -type Foo

type Foo uint
const (
    FooUnset Foo = iota
    FooBar
    FooBaz
    FooBarBaz
)
```

## No JSON null values

No structure should return a value of `null` for any JSON value. If it is a
slice, initialize it with 0 length slice. If it isn't there, mark that it
wasn't there somehow, something like `UnknownBool` works for booleans for
example. `null` values in the JSON structure will be considered a bug.

This rule is intended to enable third parties to interact with the data 
`inzure` gathers without having to litter their code with `null` checks or 
wonder what `null` means in a specific context.
