# Inzure Query Strings

Inzure Query Strings provide a method to query data quickly and perform basic tests with just a string. They form their only mini query "language" for inzure data, and can be very useful. The main uses for query strings are:

1. Pass simple identifiers between programs
2. Query inzure data directly using conditionals

## Simple Identifiers

In the simple identifier case, you'll typically be using the simplest version of a query string (newlines added for clarity):

```
/ResourceGroupField
    [/ResourceGroupName
        [/ResourceName]]
```

Only the first `ResourceGroupField` is required: adding the optional pieces means more specificity. A fully qualified query string will always return a single resource only if the data is from a real Azure subscription.

You can also take these one more level:

```
/ResourceGroupField
    [/ResourceGroupName
        [/ResourceName
            [/SubresourceField
                [/SubresourceName]]]]
```

`ResourceGroupName` and `ResourceName` can be `*` if they are not important in this case.

## Query Data

Query strings are much more than just identifiers though. They also support conditionals on the `ResourceGroupField` or `SubresourceField` entries. Conditionals are specified as follows:

```
/ResourceGroupField[FIELD OP VAL]
```

Where `FIELD` can be a chain of fields identifiers that would look very similar to the Go code for selecting things. For example,

```
/SQLServers[.FQDN LIKE ".*public.*"]
```

adds a conditional onto the selection of all SQLServers. Fields can also select slices and have a special syntax for that:

```
/SQLServers[.Firewall[0].AllowsAllAzure != BoolFalse]
/SQLServers[.Firewall[ANY].AllowsAllAzure != BoolFalse]
/SQLServers[.Firewall[ALL].AllowsAllAzure != BoolFalse]
```

are the three potential ways to deal with an array. You can either use an index (0 in this case), the word `ANY`, or the word `ALL`. `ANY` and `ALL` are special filtering indexes. In the case of `ANY`, if any of the entries in the slice satisfy the condition, then it passes. `ALL` is the same, but only if all entries satisfy the condition.

You can also call methods in a condition:

```
/NetworkSecurityGroups[.AllowsIPToPortString("Internet", "22") != BoolFalse]
/NetworkSecurityGroups[.AllowsIPToPortString("Internet", "22")[0] != BoolFalse]
```

methods must be the last selector and you can choose which return value to check with a `[X]` after the method. By default, the 0th is checked. Errors in the method calls are checked and propagated if any are returned.


Available operators are the typical comparison operators as well as `LIKE` and `!LIKE`.

### Writing Conditions

Note that, in the above conditions, I always used `!= BoolFalse` instead of `== BoolTrue`. The reason for this is that we might be interested in the `BoolUnknown` state, but that would be ignored in the second case. Care should be taken to always make sure you are accounting for Unknown cases

## Tooling

If you use the included `inzure` binary, the `inzure search` command allows for querying with Inzure Query Strings. It also has autocomplete functionality (mosted tested in `zsh`, but the `bash` ones seem to work too) which can help remember fields to use.
