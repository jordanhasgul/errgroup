# errgroup

## Overview

`errgroup` is a Go module that provides synchronisation for goroutines running fallible functions
and aggregates any errors that occurred.

## Usage

### Creating an errgroup

As per the Go proverb, the zero value of the `errgroup.Group` is useful, and you can simply create
a `errgroup.Group` as follows and begin using it:

```go
var eg errgroup.Group
```

However, if you would like to construct an `errgroup.Group` from some configuration, you can use 
the `errgroup.New` function and supply some `errgroup.Configurer`'s:

```go
var (
    ctx, cc = errgroup.WithCancel(ctx.Background())
    lc      = errgroup.WithLimit(10)
    eg      = errgroup.New(cc, lc)
)
```

### Using an errgroup

Once you've created an `errgroup.Group`, you can begin using it by calling `errgroup.Group.Go` (or
`errgroup.Group.TryGo`). Then, `errgroup.Group.Wait` for the result:  

```go
var fs []func() error
//
// ...
//
for _, f := range fs {
    _ = eg.Go(f)
}

errs := eg.Wait()
fmt.Println(errs)
```

## Documentation

Documentation for `errgroup` can be found [here](https://pkg.go.dev/github.com/jordanhasgul/errgroup).
