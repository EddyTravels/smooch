# Smooch

This is a Go library for making bots with Smooch service.

_**Note** : This a modified version version of [EddyTravels/smooch](https://github.com/EddyTravels/smooch) library with additional features. Please refer to the original repo for the original features._

## Additional Feature

- Redis support as a centralized storage to store JWT token for supporting autoscaling environment.

## Tips

Smooch documentation: https://docs.smooch.io/rest/

## Installing

```
$ go get -u github.com/kitabisa/smooch
```

## Example

```
import (
	"os"

	"github.com/kitabisa/smooch"
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
