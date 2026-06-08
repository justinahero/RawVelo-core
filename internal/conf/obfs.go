package conf

import "fmt"

// Obfs — تنظیمات obfuscation
type Obfs struct {
	Enabled     bool   `yaml:"enabled"`
	Key         string `yaml:"key"`
	Padding     *bool  `yaml:"padding"`
	Jitter      *bool  `yaml:"jitter"`
	Camouflage  *bool  `yaml:"camouflage"`
	AdaptiveFEC *bool  `yaml:"adaptive_fec"`
}

func (o *Obfs) IsPadding() bool     { return o.Padding != nil && *o.Padding }
func (o *Obfs) IsJitter() bool      { return o.Jitter != nil && *o.Jitter }
func (o *Obfs) IsCamouflage() bool  { return o.Camouflage != nil && *o.Camouflage }
func (o *Obfs) IsAdaptiveFEC() bool { return o.AdaptiveFEC != nil && *o.AdaptiveFEC }

func boolPtr(b bool) *bool { return &b }

func (o *Obfs) setDefaults() {
	if !o.Enabled {
		return
	}
	if o.Padding == nil {
		o.Padding = boolPtr(true)
	}
	if o.Jitter == nil {
		o.Jitter = boolPtr(false)
	}
	if o.Camouflage == nil {
		o.Camouflage = boolPtr(false)
	}
	if o.AdaptiveFEC == nil {
		o.AdaptiveFEC = boolPtr(true)
	}
}

// validate — اگه obfs فعاله ولی key نداره، error بده
func (o *Obfs) validate() []error {
	var errs []error
	if o.Enabled && o.Key == "" {
		errs = append(errs, fmt.Errorf("obfs.key is required when obfs is enabled"))
	}
	return errs
}
