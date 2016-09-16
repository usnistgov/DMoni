package detector

import (
	"github.com/lizhongz/dmoni/common"
)

type Detector interface {
	Detect(jobId string) ([]common.Process, error)
}
