module github.com/robbydyer/sports

go 1.15

require (
	github.com/go-co-op/gocron v0.5.1
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20201108214237-06ea97f0c265 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/markbates/pkger v0.17.1
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
	github.com/robbydyer/rgbmatrix-rpi v0.5.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.5.1
	golang.org/x/image v0.0.0-20201208152932-35266b937fa6
	golang.org/x/sys v0.0.0-20210119212857-b64e53b001e4 // indirect
	gopkg.in/yaml.v2 v2.2.8
)

replace github.com/robbydyer/rgbmatrix-rpi => ../rgbmatrix-rpi
