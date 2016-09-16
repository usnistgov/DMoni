package common

// Application info used for monitoring
type App struct {
	// Application Id
	Id string
	// Frameworks used by this application
	Frameworks []string
	// Job Ids in corresponding frameworks
	JobIds []string
	// IP address of the node where the app started
	EntryNode string
	// Pid of the application's main process
	EntryPid int32
	// Status of the application: running, exited
	Status string
}
