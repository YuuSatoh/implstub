# implstub
Selecting interface and struct will result in a temporary implementation.

## install
### Go version < 1.16
```sh
$ go get github.com/YuuSatoh/implstub
```

### Go 1.16+
```sh
$ go install github.com/YuuSatoh/implstub/cmd/implstub@latest
```


## How to use
```
USAGE:
   implstub [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --file value, -f value  specify the output file path
   --overwrite, -w         overwrite the specified receiver file (default: false)
   --pointer, -p           create a stub with the pointer receiver (default: false)
```

