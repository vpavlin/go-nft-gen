package generate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"path"
	"sync"

	"github.com/disintegration/imaging"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/mroth/weightedrand"
	"github.com/sirupsen/logrus"
	"github.com/vpavlin/go-nft-gen/internal/config"
)

const (
	MAX_RETRIES     = 10000
	MAX_NFT_RETRIES = 100
	METADATA_DIR    = "metadata"
	IMAGE_DIR       = "images"
	RARITIES_FILE   = "rarities.json"
)

var ErrNftFailed = fmt.Errorf("failed to produce valid NFT")

type Attribute struct {
	Name  string `json:"trait_type"`
	Value string `json:"value"`
}
type NFT struct {
	id          uint
	Image       string      `json:"image"`
	Attributes  []Attribute `json:"attributes"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	ExternalURL string      `json:"external_url,omitempty"`
}

type Generate struct {
	nftHash    map[string]*NFT
	AttrMax    map[Attribute]int
	AttrCount  map[Attribute]int
	config     config.Config
	choosers   map[string]weightedrand.Chooser
	ipfs       *shell.Shell
	saveCh     chan *NFT
	waitCh     chan struct{}
	progressCh chan struct{}
	waitGroup  sync.WaitGroup
}

func NewGenerate(c config.Config) (*Generate, error) {
	g := &Generate{
		AttrMax:    make(map[Attribute]int),
		AttrCount:  make(map[Attribute]int),
		nftHash:    make(map[string]*NFT),
		config:     c,
		choosers:   map[string]weightedrand.Chooser{},
		saveCh:     make(chan *NFT),
		progressCh: make(chan struct{}, 20),
		waitCh:     make(chan struct{}, c.Concurrency),
	}

	err := g.Init()
	return g, err
}

func (g *Generate) Init() error {
	if _, err := os.Stat(g.ImagesDir()); os.IsNotExist(err) {
		err := os.MkdirAll(g.ImagesDir(), 0755)
		if err != nil {
			return err
		}
	}
	if _, err := os.Stat(g.MetadataDir()); os.IsNotExist(err) {
		err = os.MkdirAll(g.MetadataDir(), 0755)
		if err != nil {
			return err
		}
	}
	for _, l := range g.config.Layers {
		choices := []weightedrand.Choice{}

		for i, v := range l.Values {
			a := Attribute{Name: l.Name, Value: v}
			cnt := g.MaxAttribute(a)
			if cnt < 0 {
				return fmt.Errorf("faild to calculate trait value percentage for %s %s", l.Name, v)
			}
			g.AttrMax[a] = cnt

			choice := weightedrand.NewChoice(i, uint(cnt))
			choices = append(choices, choice)
		}

		chooser, err := weightedrand.NewChooser(choices...)
		if err != nil {
			return err
		}

		g.choosers[l.Name] = *chooser
	}

	if g.config.IPFS != nil {
		var err error
		g.ipfs, err = g.newIpfsClient()
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generate) Max() int {
	max := 1
	for _, l := range g.config.Layers {
		max *= len(l.Values)
	}

	return max
}

func (g *Generate) MaxAttribute(a Attribute) int {
	max := float64(g.Max())
	for _, l := range g.config.Layers {
		if l.Name == a.Name {
			for i, v := range l.Values {
				if v == a.Value {
					percent := l.Weights[i]
					res := int(math.Round(max*percent) / 100)
					if res == 0 && percent > 0 {
						res = 1
					}

					return res
				}
			}
		}
	}

	return -1
}

func (g *Generate) Exists(n NFT) bool {
	_, ok := g.nftHash[n.Hash()]

	return ok
}

func (g *Generate) NewNFT() (*NFT, error) {

	nftRetries := 0
	for nftRetries < MAX_NFT_RETRIES {
		n := NFT{id: uint(len(g.nftHash))}
		nftAttrCnt := make(map[Attribute]int)
		nftRetries++
		for _, l := range g.config.Layers {
			retries := 0

			if (l.MaxId > 0 && n.id > l.MaxId) || (l.MinId > 0 && n.id < l.MinId) {
				logrus.Infof("Skipping layer %s for tokenId %d", l.Name, n.id)
				continue
			}

			for retries < MAX_RETRIES {
				retries++
				i := g.choosers[l.Name].Pick().(int)

				if (l.MaxIds != nil && (*l.MaxIds)[i] > 0 && n.id > (*l.MaxIds)[i]) || (l.MinIds != nil && n.id < (*l.MinIds)[i]) {
					continue
				}

				value := l.Values[i]

				a := Attribute{
					Name:  l.Name,
					Value: value,
				}

				cnt := g.AttrCount[a] + 1
				if cnt > g.AttrMax[a] {
					continue
				}

				n.Attributes = append(n.Attributes, a)
				nftAttrCnt[a] = cnt
				break
			}

		}

		if g.Exists(n) {
			logrus.Debugf("NFT with the same signature exists, retrying (id: %d)", n.id)
			continue
		}
		g.nftHash[n.Hash()] = &n
		for key, val := range nftAttrCnt {
			g.AttrCount[key] = val
		}

		n.Name = fmt.Sprintf(g.config.NameFmtTmplt, n.id)
		if g.config.ExternalURLTmplt != "" {
			n.ExternalURL = fmt.Sprintf(g.config.ExternalURLTmplt, n.id)
		}

		n.Description = g.config.Description

		return &n, nil
	}

	return &NFT{}, ErrNftFailed
}

func (g *Generate) getImage(a Attribute) string {
	for _, l := range g.config.Layers {
		if l.Name == a.Name {
			for i, v := range l.Values {
				if v == a.Value {
					filename := fmt.Sprintf("%s.png", l.Images[i])
					return path.Join(l.BasePath, filename)
				}
			}
		}
	}

	return ""
}

func (g *Generate) SetImagePath(n *NFT, image string) error {
	n.SetImagePath(image)

	return nil
}

func (g *Generate) Image(n *NFT) error {

	var base image.Image
	for _, a := range n.Attributes {
		img := g.getImage(a)
		image, err := imaging.Open(img)
		if err != nil {
			return err
		}
		if base == nil {
			base = image
			continue
		}

		base = imaging.OverlayCenter(base, image, 1.0)
	}

	if base == nil {
		return nil
	}

	imagePath := path.Join(g.ImagesDir(), fmt.Sprintf("%d.png", n.id))

	err := imaging.Save(base, imagePath)
	if err != nil {
		return err
	}

	g.SetImagePath(n, imagePath)

	return nil

}

func (g *Generate) Metadata(n *NFT) error {
	metadataPath := path.Join(g.MetadataDir(), fmt.Sprintf("%d.json", n.id))

	data, err := json.MarshalIndent(n, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metadataPath, data, 0644)
}

func (g *Generate) GenerateN(n uint) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go g.Save(ctx)
	go g.Progress(ctx, n)
	for i := uint(0); i < n; i++ {
		n, err := g.NewNFT()
		if err != nil {
			logrus.Errorf("NFT %d failed: %s", i, err)

			if errors.Is(err, ErrNftFailed) {
				logrus.Errorf("Stopping")
				break
			}
			continue
		}

		g.waitCh <- struct{}{}
		g.saveCh <- n
	}

	logrus.Debugf("Waiting for processing to finish")
	g.waitGroup.Wait()

	return nil
}

func (g *Generate) Save(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case n := <-g.saveCh:
			g.waitGroup.Add(1)
			go func(n *NFT) {
				defer g.waitGroup.Done()
				defer func() { <-g.waitCh }()
				err := g.Image(n)
				if err != nil {
					logrus.Error(fmt.Errorf("failed to save image for NFT %d: %s", n.id, err))
					return
				}

				err = g.Metadata(n)
				if err != nil {
					logrus.Error(fmt.Errorf("failed to save metadata for NFT %d: %s", n.id, err))
					return
				}

				logrus.Debugf("Done with NFT %d", n.id)
				g.progressCh <- struct{}{}
			}(n)
		}
	}

}

func (g *Generate) Progress(ctx context.Context, n uint) {
	progress := 0
	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			return
		case <-g.progressCh:
			progress++
			fmt.Printf("\r => Processed %d/%d NFTs", progress, n)
		}
	}
}

type AttributeRarity struct {
	PercentTotal float64
	Percent      float64
	Count        int
}

func (g *Generate) Rarities() map[string]map[string]AttributeRarity {
	result := make(map[string]map[string]AttributeRarity)
	totals := make(map[string]float64)
	attrs := make([]Attribute, 0)
	for _, l := range g.config.Layers {
		for _, v := range l.Values {
			a := Attribute{Name: l.Name, Value: v}
			cnt := g.AttrCount[a]

			if _, ok := totals[a.Name]; !ok {
				totals[a.Name] = 0
			}

			totals[a.Name] += float64(cnt)
			attrs = append(attrs, a)
		}
	}

	for _, a := range attrs {
		cnt := g.AttrCount[a]
		if _, ok := result[a.Name]; !ok {
			result[a.Name] = map[string]AttributeRarity{}
		}

		ar := AttributeRarity{
			PercentTotal: float64(cnt*100) / float64(g.NftCount()),
			Percent:      float64(cnt*100) / totals[a.Name],
			Count:        cnt,
		}

		result[a.Name][a.Value] = ar
	}

	return result
}

func (g *Generate) WriteRarities() error {
	rarities := g.Rarities()
	data, err := json.MarshalIndent(rarities, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(g.RaritiesFile(), data, 0644)

	return err
}

func (g *Generate) MetadataDir() string {
	return path.Join(g.config.OutputDir, METADATA_DIR)
}

func (g *Generate) ImagesDir() string {
	return path.Join(g.config.OutputDir, IMAGE_DIR)
}

func (g *Generate) RaritiesFile() string {
	return path.Join(g.config.OutputDir, RARITIES_FILE)
}

func (g *Generate) NftCount() int {
	return len(g.nftHash)
}

//NFT funcs

func NewNFT() NFT {
	return NFT{
		Attributes: make([]Attribute, 0),
	}
}

func (n *NFT) Hash() string {
	hash := ""
	for _, a := range n.Attributes {
		hash = fmt.Sprintf("%s:%s=%s", hash, a.Name, a.Value)
	}

	return hash
}

func (n *NFT) SetImagePath(image string) {
	n.Image = image
}
