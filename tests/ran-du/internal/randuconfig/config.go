package randuconfig

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gosystem/tests/internal/config"
	"gopkg.in/yaml.v2"
)

const (
	// PathToDefaultRanDuParamsFile path to config file with default ran du parameters.
	PathToDefaultRanDuParamsFile = "./default.yaml"
)

// RanDuConfig type keeps ran du configuration.
type RanDuConfig struct {
	*config.GeneralConfig
	TestWorkload struct {
		Namespace      string `yaml:"namespace" envconfig:"ECO_RANDU_TESTWORKLOAD_NAMESPACE"`
		CreateMethod   string `yaml:"create_method" envconfig:"ECO_RANDU_TESTWORKLOAD_CREATE_METHOD"`
		CreateShellCmd string `yaml:"create_shell_cmd" envconfig:"ECO_RANDU_TESTWORKLOAD_CREATE_SHELLCMD"`
	} `yaml:"randu_test_workload"`
	SoftRebootIterations     string `yaml:"soft_reboot_iterations" envconfig:"ECO_RANDU_SOFT_REBOOT_ITERATIONS"`
	HardRebootIterations     string `yaml:"hard_reboot_iterations" envconfig:"ECO_RANDU_HARD_REBOOT_ITERATIONS"`
	IpmiToolImage            string `yaml:"ipmitool_image" envconfig:"ECO_RANDU_IPMITOOL_IMAGE"`
	LaunchWorkloadIterations string `yaml:"launch_workload_iterations" envconfig:"ECO_RANDU_LAUNCH_WORKLOAD_ITERATIONS"`
}

// NewRanDuConfig returns instance of RanDuConfig config type.
func NewRanDuConfig() *RanDuConfig {
	log.Print("Creating new RanDuConfig struct")

	var randuConf RanDuConfig
	randuConf.GeneralConfig = config.NewConfig()

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)
	confFile := filepath.Join(baseDir, PathToDefaultRanDuParamsFile)
	err := readFile(&randuConf, confFile)

	if err != nil {
		log.Printf("Error to read config file %s", confFile)

		return nil
	}

	err = readEnv(&randuConf)

	if err != nil {
		log.Print("Error to read environment variables")

		return nil
	}

	return &randuConf
}

func readFile(randuConfig *RanDuConfig, cfgFile string) error {
	openedCfgFile, err := os.Open(cfgFile)
	if err != nil {
		return err
	}

	defer func() {
		_ = openedCfgFile.Close()
	}()

	decoder := yaml.NewDecoder(openedCfgFile)
	err = decoder.Decode(&randuConfig)

	if err != nil {
		return err
	}

	return nil
}

func readEnv(randuConfig *RanDuConfig) error {
	err := envconfig.Process("", randuConfig)
	if err != nil {
		return err
	}

	return nil
}
