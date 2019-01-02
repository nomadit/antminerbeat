package scan

type job struct {
	id       int64
	mac      string
	ip       string
	status   string
	user     *string
	password *string
	hostname *string
	isValid  bool
	finish   bool
	first    bool
}

