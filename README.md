# League Backend Challenge

Go service for operations on square CSV matrices of integers. Requires Go 1.23 or newer; no external dependencies.

## Run

```sh
go run .
```

Send a multipart `POST` with the CSV in the `file` field. The included `matrix.csv` can be used to try each endpoint:

```sh
curl -F 'file=@matrix.csv' http://localhost:8080/echo
curl -F 'file=@matrix.csv' http://localhost:8080/invert
curl -F 'file=@matrix.csv' http://localhost:8080/flatten
curl -F 'file=@matrix.csv' http://localhost:8080/sum
curl -F 'file=@matrix.csv' http://localhost:8080/multiply
```

`echo` returns the matrix, `invert` transposes it, and `flatten` lists its values in row order. `sum` and `multiply` use arbitrary-precision integers. For the included matrix, they return `45` and `362880`.

Input must be a non-empty square matrix with no header. Invalid input returns 400, non-POST requests return 405, and requests over 10 MiB return 413.

## Test

```sh
go test ./...
```
