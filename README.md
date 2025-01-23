## Convert

A Go package for converting between different units of measurement.

### Installation

```
go get github.com/carlwf/convert
```

### Usage

Basic Conversion example:

```
val, err := convert.Value(123.45,"Celsius","Fahrenheit")
```
JSON example:
``` 
jsonbytes, err := convert.ToJson((123.45,"Celsius","Fahrenheit")
```






