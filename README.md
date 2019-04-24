# Smooch

This is a Go library for making bots with Smooch service.

## Tips

Smooch documentation: https://docs.smooch.io/rest/

## Installing

```
$ go get -u github.com/EddyTravels/smooch
```

## Example

```
import (
	"os"

	"github.com/EddyTravels/smooch"
)

func main() {
    smoochClient, err := smooch.New(smooch.Options{
        AppID:        os.Getenv("SMOOCH_APP_ID"),
        KeyID:        os.Getenv("SMOOCH_KEY_ID"),
        Secret:       os.Getenv("SMOOCH_SECRET"),
        VerifySecret: os.Getenv("SMOOCH_VERIFY_SECRET"),
    })

    if err != nil {
        panic(err)
    }
}
```

## Contributing
You are more than welcome to contribute to this project. Fork and make a Pull Request, or create an Issue if you see any problem.
