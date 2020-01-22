# go-gqlgen-helpers ![](https://github.com/nrfta/go-gqlgen-helpers/workflows/CI/badge.svg)

## Install

```sh
go get -u "github.com/nrfta/go-gqlgen-helpers"
```



### GraphQL gqlgen error handling


```go

// ...
router.Handle("/graphql",
  handler.GraphQL(
    NewExecutableSchema(Config{Resolvers: &Resolver{}}),
    errorhandling.ConfigureErrorPresenterFunc(reportErrorToSentry),
    errorhandling.ConfigureRecoverFunc(),
  ),
)

// ...

func reportErrorToSentry(ctx context.Context, err error) {
  // Whatever
}

```

## License

This project is licensed under the [MIT License](LICENSE.md).
