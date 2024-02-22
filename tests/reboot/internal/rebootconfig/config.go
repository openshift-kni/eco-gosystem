package rebootconfig

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gosystem/tests/internal/config"
	systemtestsparams "github.com/openshift-kni/eco-gosystem/tests/internal/params"
	"gopkg.in/yaml.v2"
)

const (
	// PathToDefaultRebootParamsFile path to config file with default ran du parameters.
	PathToDefaultRebootParamsFile = "./default.yaml"
)

// RebootConfig type keeps ran du configuration.
type RebootConfig struct {
	*config.GeneralConfig
	NodesCredentialsMap systemtestsparams.NodesBMCMap `yaml:"nodes_bmc_map" envconfig:"ECO_SYSTEM_NODES_CREDENTIALS_MAP"`
	//
	ControlPlaneLabelStr string `yaml:"control_plane_nodes_label" envconfig:"ECO_REBOOT_CONTROL_PLANE_NODES_LABEL"`
	MasterNodesLabelStr  string `yaml:"master_nodes_label" envconfig:"ECO_REBOOT_MASTER_NODES_LABEL"`
	WorkerNodesLabelStr  string `yaml:"worker_nodes_label" envconfig:"ECO_REBOOT_WORKER_NODES_LABEL"`
}

// NewRebootConfig returns instance of RebootConfig config type.
func NewRebootConfig() *RebootConfig {
	log.Print("Creating new RebootConfig struct")

	var rebootConf RebootConfig
	rebootConf.GeneralConfig = config.NewConfig()

	var confFile string

	if fileFromEnv, exists := os.LookupEnv("ECO_SYSTEM_REBOOT_CONFIG_FILE_PATH"); !exists {
		_, filename, _, _ := runtime.Caller(0)
		baseDir := filepath.Dir(filename)
		confFile = filepath.Join(baseDir, PathToDefaultRebootParamsFile)
	} else {
		confFile = fileFromEnv
	}

	log.Printf("Open config file %s", confFile)

	err := readFile(&rebootConf, confFile)
	if err != nil {
		log.Printf("Error to read config file %s", confFile)

		return nil
	}

	err = readEnv(&rebootConf)

	if err != nil {
		log.Print("Error to read environment variables")

		return nil
	}

	return &rebootConf
}

func readFile(rebootConfig *RebootConfig, cfgFile string) error {
	openedCfgFile, err := os.Open(cfgFile)
	if err != nil {
		return err
	}

	defer func() {
		_ = openedCfgFile.Close()
	}()

	decoder := yaml.NewDecoder(openedCfgFile)
	err = decoder.Decode(&rebootConfig)

	if err != nil {
		return err
	}

	return nil
}

func readEnv(rebootConfig *RebootConfig) error {
	err := envconfig.Process("", rebootConfig)
	if err != nil {
		return err
	}

	return nil
}
