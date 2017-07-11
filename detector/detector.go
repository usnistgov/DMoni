package detector

import (
	"github.com/usnistgov/DMoni/common"
)

type Detector interface {
	Detect(jobId string) ([]common.Process, error)
}
