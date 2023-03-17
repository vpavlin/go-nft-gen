package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	DEFAULT_OUTPUT_DIR    = "./output"
	DEFAULT_NAME_TEMPLATE = "NFT #%d"
)

type Config struct {
	N                uint    `json:"n"`
	Layers           []Layer `json:"layers"`
	OutputDir        string  `json:"output_dir"`
	NameFmtTmplt     string  `json:"name_format_template"`
	ExternalURLTmplt string  `json:"external_url_template"`
	Description      string  `json:"description"`
	IPFS             *IPFS   `json:"ipfs"`
	Concurrency      int     `json:"concurrency"`
}

type Layer struct {
	Name     string    `json:"name"`
	BasePath string    `json:"base_path"`
	MinId    uint      `json:"min_id"`
	MaxId    uint      `json:"max_id"`
	Images   []string  `json:"images"`
	Values   []string  `json:"values"`
	Weights  []float64 `json:"weights"`
	MinIds   *[]uint   `json:"min_ids"`
	MaxIds   *[]uint   `json:"max_ids"`
}

type IPFS struct {
	Endpoint      string `json:"endpoint"`
	ProjectID     string `json:"project_id"`
	ProjectSecret string `json:"project_secret"`
}

func Load(filename string) (Config, error) {
	c := Config{}
	data, err := os.ReadFile(filename)
	if err != nil {
		return c, err
	}

	err = json.Unmarshal(data, &c)
	if err != nil {
		return c, err
	}

	for i, l := range c.Layers {
		if len(l.Images) == 0 {
			logrus.Debugf("No images defined for %s, loading from dir %s", l.Name, l.BasePath)
			c.Layers[i].Images = make([]string, 0)
			files, err := os.ReadDir(l.BasePath)
			if err != nil {
				return c, err
			}

			for _, f := range files {
				if f.IsDir() {
					continue
				}
				ext := path.Ext(f.Name())
				if ext != ".png" {
					continue
				}

				c.Layers[i].Images = append(c.Layers[i].Images, strings.TrimSuffix(f.Name(), ext))
			}
		}
	}

	for i, l := range c.Layers {
		if len(l.Values) == 0 && len(l.Images) > 0 {
			logrus.Debugf("No values specified for %s, using image names", l.Name)
			c.Layers[i].Values = l.Images
		}

	}
	for i, l := range c.Layers {
		if len(l.Weights) == 0 {
			w := float64(100) / float64(len(l.Values))
			logrus.Debugf("No weights specified for %s, using equal weights of %f", l.Name, w)

			c.Layers[i].Weights = make([]float64, len(l.Values))
			for j, _ := range l.Values {
				c.Layers[i].Weights[j] = w
			}
		} else {
			sum := 0.0
			for _, w := range l.Weights {
				sum += w
			}

			if sum != 100 {
				return c, fmt.Errorf("wights sum in layer %s is not 100: %f", l.Name, sum)
			}
		}
	}

	if c.NameFmtTmplt == "" {
		logrus.Debugf("No name template specified, using default template '%s'", DEFAULT_NAME_TEMPLATE)
		c.NameFmtTmplt = DEFAULT_NAME_TEMPLATE
	}
	if c.OutputDir == "" {
		logrus.Debugf("No output dir specified, using default output dir %s", DEFAULT_OUTPUT_DIR)
		c.OutputDir = DEFAULT_OUTPUT_DIR
	}

	if c.Concurrency == 0 {
		cpus := runtime.NumCPU()
		logrus.Debugf("No concurrency specified, using number of CPUs %d", cpus)
		c.Concurrency = cpus
	}

	return c, nil
}
