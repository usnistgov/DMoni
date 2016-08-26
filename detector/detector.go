package detector

type Process struct {
	Pid       string
	ShortName string
	FullName  string
}

type Detector interface {
	Detect(appId string) ([]Process, error)
}
