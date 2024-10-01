package cfg

import "github.com/spf13/viper"

type Config struct {
	TenantID             string `mapstructure:"TENANT_ID"`
	ClientID             string `mapstructure:"CLIENT_ID"`
	ClientSecret         string `mapstructure:"CLIENT_SECRET"`
	SubscriptionID       string `mapstructure:"SUBSCRIPTION_ID"`
	RemoteSubscriptionID string `mapstructure:"REMOTE_SUBSCRIPTION_ID"`
	HubRGName            string `mapstructure:"HUB_RG_NAME"`
	HubVnetName          string `mapstructure:"HUB_VNET_NAME"`
	SpokeRGName          string `mapstructure:"SPOKE_RG_NAME"`
	SpokeVnetName        string `mapstructure:"SPOKE_VNET_NAME"`
	SpokeRouteTableName  string  `mapstructure:"SPOKE_ROUTE_TABLE_NAME"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	err = viper.ReadInConfig()
	if err != nil {
		return
	}
	err = viper.Unmarshal(&config)
	return
}
