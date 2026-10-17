# League Backend Challenge

This Go HTTP service performs five operations on an uploaded square CSV matrix of integers. It uses only the Go standard library (Go 1.23 or newer).

Given an uploaded csv file
```
1,2,3
4,5,6
7,8,9
```

1. Echo
    - Return the matrix as a string in matrix format.
    
    ```
    // Expected output
    1,2,3
    4,5,6
    7,8,9
    ``` 
2. Invert
    - Return the matrix as a string in matrix format where the columns and rows are inverted
    ```
    // Expected output
    1,4,7
    2,5,8
    3,6,9
    ``` 
3. Flatten
    - Return the matrix as a 1 line string, with values separated by commas.
    ```
    // Expected output
    1,2,3,4,5,6,7,8,9
    ``` 
4. Sum
    - Return the sum of the integers in the matrix
    ```
    // Expected output
    45
    ``` 
5. Multiply
    - Return the product of the integers in the matrix
    ```
    // Expected output
    362880
    ``` 

The input file to these functions is a matrix, of any dimension where the number of rows are equal to the number of columns (square). Each value is an integer, and there is no header row. matrix.csv is example valid input.  

Run the server (port 8080)
```
go run .
```

Send request
```
curl -F 'file=@/path/matrix.csv' "localhost:8080/echo"
curl -F 'file=@/path/matrix.csv' "localhost:8080/invert"
curl -F 'file=@/path/matrix.csv' "localhost:8080/flatten"
curl -F 'file=@/path/matrix.csv' "localhost:8080/sum"
curl -F 'file=@/path/matrix.csv' "localhost:8080/multiply"
```

Each endpoint accepts a multipart `POST` with a `file` field. The input must have no header and contain an integer in every cell. `echo`, `invert`, and `flatten` retain the uploaded cell text; `echo` and `invert` serialize it as CSV. `sum` and `multiply` use arbitrary-precision integers to avoid overflow.

Invalid input returns HTTP 400 with an error message. Unsupported methods return HTTP 405, and requests over 10 MiB (including multipart overhead) return HTTP 413.

Run the unit tests with `go test ./...`.
