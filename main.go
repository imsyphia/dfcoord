package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
)

type twoParams struct {
	okx, okz bool
	x, z     dfParams
}

// the file writing is a hack for now

func main() {
	var err error
	cpufile, err := os.Create("cpu.pprof")
	if err != nil {
		log.Fatal(err)
	}

	err = pprof.StartCPUProfile(cpufile)
	if err != nil {
		log.Fatal(err)
	}

	defer cpufile.Close()
	defer pprof.StopCPUProfile()

	if len(os.Args) < 2 || len(os.Args) > 2 {
		log.Fatal("Usage: dfcoord <dimension seed>")
	}

	dimSeed, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	reduce := func(a twoParams, first bool, d dfParams) (twoParams, bool) {
		if d.axis == axisX {
			if !a.okx {
				a.x = d
				a.okx = true
			}
		} else {
			if !a.okz {
				a.z = d
				a.okz = true
			}
		}
		cont := !(a.okx && a.okz)
		return a, cont
	}

	t := genFromDimSeed(int64(dimSeed), reduce)

	err = os.MkdirAll(fmt.Sprintf("%s/worldgen/density_function", namespace), fs.ModeDir+fs.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(fmt.Sprintf("%s/worldgen/noise", namespace), fs.ModeDir+fs.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	err = writeNoiseFile(split(t.x.rl))
	if err != nil {
		log.Fatal(err)
	}

	err = writeNoiseFile(split(t.z.rl))
	if err != nil {
		log.Fatal(err)
	}

	err = writeDfFile(namespace, "x", t.x)
	if err != nil {
		log.Fatal(err)
	}

	err = writeDfFile(namespace, "z", t.z)
	if err != nil {
		log.Fatal(err)
	}

	return

}

func writeDfFile(ns string, name string, p dfParams) error {
	s := fmt.Sprintf(dfFormat, p.b, p.m, p.rl, p.x, p.y, p.z)
	return os.WriteFile(fmt.Sprintf("%s/worldgen/density_function/%s.json", ns, name), []byte(s), 0644)
}

func writeNoiseFile(ns string, name string) error {
	return os.WriteFile(fmt.Sprintf("%s/worldgen/noise/%s.json", ns, name), []byte(noiseFile), 0644)
}

func split(rl string) (namespace string, id string) {
	s := strings.Split(rl, ":")
	return s[0], s[1]
}

var noiseFile = `{
    "firstOctave": 0,
    "amplitudes": [
        1.0
    ]
}`

var dfFormat = `{
	"type": "minecraft:flat_cache",
	"argument": {
		"type": "minecraft:mul",
		"argument1": {
			"type": "minecraft:mul",
			"argument1": 1.0e6,
			"argument2": 1.0e3
		},
		"argument2": {
			"type": "minecraft:add",
			"argument1": %.14g,
			"argument2": {
				"type": "minecraft:mul",
				"argument1": %.14g,
				"argument2": {
					"type": "minecraft:shifted_noise",
					"noise": "%s",
					"xz_scale": 1.0e-9,
					"y_scale": 0.0,
					"shift_x": %.14g,
					"shift_y": %.14g,
					"shift_z": %.14g
				}
			}
		}
	}
}`
