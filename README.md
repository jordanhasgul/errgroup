# errgroup

## Overview

`errgroup` is a Go module that provides Context cancellation, error propagation and synchronisation 
for goroutines running fallible functions.

## Usage

### Creating an errgroup

As per the Go proverb, the zero value of the `errgroup.Group` is useful, and you can simply create
a `errgroup.Group` as follows and begin using it:

```go
var eg errgroup.Group
```

You can also configure the behaviour of an `errgroup.Group` by providing some `errgroup.Configurer`'s to the 
`errgroup.New` function. For example:

```go
var (
    ctx, cc = errgroup.WithCancel(ctx.Background())
    eg      = errgroup.New(cc)
)
```

### Using an errgroup

Once you have created an `errgroup.Group`, you can begin using it by calling `errgroup.Group.Go`. Then, 
you can call `errgroup.Group.Wait` to wait for the result:  

```go
var fs []func() error

// ...

for _, f := range fs {
    _ = eg.Go(f)
}

errs := eg.Wait()
fmt.Println(errs)
```

## Documentation

Documentation for `errgroup` can be found [here](https://pkg.go.dev/github.com/jordanhasgul/errgroup).
