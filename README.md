# go-nft-gen

A simple NFT generation tool written in Golang. It allows you to generate images from provided layers and variants based on configuration preferences

## Configuration

Minimal configuration could look like this:

```
{
    "description": "These are just some fancy NFTs to show off my amazing design skills",
    "layers": [
        {
            "name": "Background",
            "base_path": "./trait-layers/backgrounds"
        },
        {
            "name": "Left-top",
            "base_path": "./trait-layers/left-top"
        }
    ]
}
```

This configuration will produce images and NFT metadata based on the PNG files found in the `base_path` folders. It will use the name of the file (minus the extension) as the trait values and set equal weights for all the traits.

### Layer

```
{
    "name": "Right-bottom",
    "min_id": 10,
    "max_id": 40,
    "base_path": "./trait-layers/right-bottom",
    "images": [
        "x",
        "y",
        "z"
    ],
    "values": [
        "one", "two", "three"
    ],
    "weights": [
        60, 30, 10
    ],
    "min_ids": [
        10, 20, 30
    ],
    "max_ids": [
        20, 30, 0
    ]
}
```

The above layer definition lists all the potential fields you can configure for each layer/trait.

* `name` - Trait name (user as `trait_type` in OpenSea compatible metadata)
* `min_id` - token id when to start applying the layer
* `max_id` - token id when to stop applying the layer
* `base_path` - folder containing the layer PNG files
* `images` - list of file names (without extension) to use for the layer
* `values` - string values used to represent the trait variant in the metadata
* `weights` - probability to pick the trait variant in %, sum of all weights must be equal 100
* `min_ids` - token id above which to allow a particular variant
* `max_ids` - token id above which to deny a particular variant

## Execute

## Build

```
make build
make install
nftgen -v
```