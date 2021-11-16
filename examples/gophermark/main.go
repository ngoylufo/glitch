package main

import (
	"fmt"
	"log"
	"embed"
	"image"
	"image/draw"
	_ "image/png"
	"time"
	"math/rand"
	"runtime"
	"runtime/pprof"
	"flag"
	"os"

	"github.com/ungerik/go3d/vec3"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/jstewart7/glitch"
	"github.com/jstewart7/glitch/shaders"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

//go:embed gopher.png
var f embed.FS
func loadImage(path string) (*image.NRGBA, error) {
	file, err := f.Open(path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	nrgba := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), img, bounds.Min, draw.Src)
	return nrgba, nil
}

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		go func() {
			time.Sleep(10 * time.Second)
			if err := pprof.StartCPUProfile(f); err != nil {
				log.Fatal("could not start CPU profile: ", err)
			}
		}()
		defer pprof.StopCPUProfile()
	}

	glitch.Run(runGame)

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}

func runGame() {
	win, err := glitch.NewWindow(1920, 1080, "Glitch", glitch.WindowConfig{
		Vsync: false,
	})
	if err != nil { panic(err) }

	shader, err := glitch.NewShader(shaders.SpriteShader)
	if err != nil { panic(err) }

	pass := glitch.NewRenderPass(shader)

	manImage, err := loadImage("gopher.png")
	if err != nil {
		panic(err)
	}
	texture := glitch.NewTexture(160, 200, manImage.Pix)
	pass.SetTexture(0, texture)

	// mesh := glitch.NewQuadMesh()
	x := float32(0)
	y := float32(0)
	manSprite := glitch.NewSprite(texture, glitch.R(x, y, x+160, y+200))

	length := 100000
	man := make([]Man, length)
	for i := range man {
		man[i] = NewMan()
	}

	w := float32(160.0)/4
	h := float32(200.0)/4
	manSize := &vec3.T{w, h, 1}

	camera := glitch.NewCamera()
	start := time.Now()
	for !win.ShouldClose() {
		if win.Pressed(glitch.KeyBackspace) {
			win.Close()
		}
		start = time.Now()
		for i := range man {
			man[i].position[0] += man[i].velocity[0]
			man[i].position[1] += man[i].velocity[1]

			if man[i].position[0] <= 0 || (man[i].position[0]+w) >= float32(1920) {
				man[i].velocity[0] = -man[i].velocity[0]
			}
			if man[i].position[1] <= 0 || (man[i].position[1]+h) >= float32(1080) {
				man[i].velocity[1] = -man[i].velocity[1]
			}
		}

		pass.Clear()

		camera.SetOrtho2D(win)
		camera.SetView2D(0, 0, 1.0, 1.0)

		for i := range man {
			mat := glitch.Mat4Ident
			mat.Scale(manSize[0], manSize[1], 1.0).Translate(man[i].position[0], man[i].position[1], 0)

			// mesh.DrawColorMask(pass, mat, glitch.RGBA{0.5, 1.0, 1.0, 1.0})
			manSprite.DrawColorMask(pass, mat, glitch.RGBA{1.0, 1.0, 1.0, 0.5})
		}

		glitch.Clear(glitch.RGBA{0.1, 0.2, 0.3, 1.0})

		pass.SetUniform("projection", camera.Projection)
		pass.SetUniform("view", camera.View)
		pass.Draw(win)

		win.Update()

		dt := time.Since(start)
		fmt.Println(dt.Seconds() * 1000)
	}
}

type Man struct {
	position, velocity mgl32.Vec2
	R, G, B float32
}
func NewMan() Man {
	vScale := 5.0
	return Man{
		// position: mgl32.Vec2{100, 100},
		// position: mgl32.Vec2{float32(float64(width/2) * rand.Float64()),
		// 	float32(float64(height/2) * rand.Float64())},
		position: mgl32.Vec2{1920/2, 1080/2},
		velocity: mgl32.Vec2{float32(2*vScale * (rand.Float64()-0.5)),
			float32(2*vScale * (rand.Float64()-0.5))},
		R: rand.Float32(),
		G: rand.Float32(),
		B: rand.Float32(),
	}
}

/*
const (
	vertexSource = `
#version 330 core
layout (location = 0) in vec3 aPos;
layout (location = 1) in vec3 aColor;
layout (location = 2) in vec2 aTexCoord;

out vec3 ourColor;
out vec2 TexCoord;

uniform mat4 projection;
uniform mat4 transform;

void main()
{
	gl_Position = projection * transform * vec4(aPos, 1.0);
//	gl_Position = vec4(aPos, 1.0);
	ourColor = aColor;
	TexCoord = vec2(aTexCoord.x, aTexCoord.y);
}
`
	fragmentSource = `
#version 330 core
out vec4 FragColor;

in vec3 ourColor;
in vec2 TexCoord;

//texture samplers
uniform sampler2D texture1;

void main()
{
	// linearly interpolate between both textures (80% container, 20% awesomeface)
	//FragColor = mix(texture(texture1, TexCoord), texture(texture2, TexCoord), 0.2);
  FragColor = vec4(ourColor, 1.0) * texture(texture1, TexCoord);
//  FragColor = vec4(ourColor, 1.0);
}
`
)*/
