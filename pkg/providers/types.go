package providers

import (
	"github.com/pibblokto/backlokto-worker/pkg/targets"
	"github.com/pibblokto/backlokto-worker/pkg/types"
)

var TargetsMap = map[string]func(map[string]string, *types.Artifacts){
	"s3":  targets.S3Target,
	"gcs": targets.GcsTarget,
}
