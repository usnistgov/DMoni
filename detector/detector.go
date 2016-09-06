package detector

import (
	"github.com/lizhongz/dmoni/common"
)

type Detector interface {
	Detect(appId string) ([]Process, error)
}
