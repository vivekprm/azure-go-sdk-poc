package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/vivekprm/azure-go-sdk-poc/pkg/cfg"
)

const (
	rtIDFormat           = "/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/routeTables/%s/routes/%s"
	vnetIDFormat         = "/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s"
	defaultAddressPrefix = "0.0.0.0/0"
)

type AuthConf struct {
	tenantID     string
	clientID     string
	clientSecret string
}

type Client struct {
	routeTablesClient  armnetwork.RouteTablesClient
	routesClient       armnetwork.RoutesClient
	vnetClient         armnetwork.VirtualNetworksClient
	vnetPeeringsClient armnetwork.VirtualNetworkPeeringsClient
}

func makeClients(cfg cfg.Config) []*Client {
	authConf := AuthConf{
		tenantID:     cfg.TenantID,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
	}

	client1 := createClientForSubscription(authConf, cfg.SubscriptionID)
	client2 := createClientForSubscription(authConf, cfg.RemoteSubscriptionID)

	return []*Client{
		client1,
		client2,
	}
}

func createClientForSubscription(authConf AuthConf, subscriptionID string) *Client {
	azureCred, err := azidentity.NewClientSecretCredential(authConf.tenantID, authConf.clientID, authConf.clientSecret, &azidentity.ClientSecretCredentialOptions{})
	if err != nil {
		fmt.Printf("Error in creating client %v", err)
		return nil
	}
	routeTablesClient, err := armnetwork.NewRouteTablesClient(subscriptionID, azureCred, &policy.ClientOptions{})
	if err != nil {
		fmt.Printf("Error in creating route tables client %v", err)
		return nil
	}
	routesClient, err := armnetwork.NewRoutesClient(subscriptionID, azureCred, &policy.ClientOptions{})
	if err != nil {
		fmt.Printf("Error in creating routes client %v", err)
		return nil
	}
	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, azureCred, &policy.ClientOptions{})
	if err != nil {
		fmt.Printf("Error in creating virtual network client %v", err)
		return nil
	}

	vnetPeeringsClient, err := armnetwork.NewVirtualNetworkPeeringsClient(subscriptionID, azureCred, &arm.ClientOptions{})
	if err != nil {
		fmt.Printf("Error in creating virtual networks peering client %v", err)
		return nil
	}
	return &Client{
		routeTablesClient:  *routeTablesClient,
		routesClient:       *routesClient,
		vnetClient:         *vnetClient,
		vnetPeeringsClient: *vnetPeeringsClient,
	}
}
func main() {
	config, err := cfg.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}
	ctx := context.Background()
	clients := makeClients(config)

	rgName := config.HubRGName
	vnetName := config.HubVnetName
	peerName1 := "peer-137749"
	peerName2 := "peer-237749"
	remoteRgName := config.SpokeRGName
	remoteVnetName := config.SpokeVnetName
	// Peering from hub to spoke
	createVnetPeeringCrossSub(ctx, clients[0], rgName, vnetName, peerName1, config.RemoteSubscriptionID, remoteRgName, remoteVnetName)
	// Peering from spoke to hub
	createVnetPeeringCrossSub(ctx, clients[1], remoteRgName, remoteVnetName, peerName2, config.SubscriptionID, rgName, vnetName)

	// rtbName := cfg.SpokeRouteTableName
	// createRouteTable(ctx, clients, rgName, rtbName)

	// rtName := "def-route"
	// createRoute(ctx, clients, config.RemoteSubscriptionID, rgName, rtbName, rtName)

	// vname := config.SpokeVnetName
	// getVnet(ctx, clients, rgName, vname)
}

func createVnetPeeringCrossSub(ctx context.Context, clients *Client, rgName, vnetName, peerName, remoteSubscription, remoteRgName, remoteVnetName string) {
	poller, err := clients.vnetPeeringsClient.BeginCreateOrUpdate(ctx, rgName, vnetName, peerName, armnetwork.VirtualNetworkPeering{
		Properties: &armnetwork.VirtualNetworkPeeringPropertiesFormat{
			AllowForwardedTraffic:     to.Ptr(true),
			AllowGatewayTransit:       to.Ptr(false),
			AllowVirtualNetworkAccess: to.Ptr(true),
			RemoteVirtualNetwork: &armnetwork.SubResource{
				ID: to.Ptr(fmt.Sprintf(vnetIDFormat, remoteSubscription, remoteRgName, remoteVnetName)),
			},
			UseRemoteGateways: to.Ptr(false),
		},
	}, &armnetwork.VirtualNetworkPeeringsClientBeginCreateOrUpdateOptions{SyncRemoteAddressSpace: to.Ptr(armnetwork.SyncRemoteAddressSpaceTrue)})
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}
	res, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		log.Fatalf("failed to pull the result: %v", err)
	}
	fmt.Printf("Peering response from %s to %s details: ID %s, Name %s, PeeringState: %s, PeeringSyncLevel: %s, Provisioning State: %s\n\n", vnetName, remoteVnetName, *res.ID, *res.Name, *res.Properties.PeeringState, *res.Properties.PeeringSyncLevel, *res.Properties.ProvisioningState)
}

func getVnet(ctx context.Context, clients *Client, rgName, vname string) {
	vnetRes, err := clients.vnetClient.Get(ctx, rgName, vname, &armnetwork.VirtualNetworksClientGetOptions{Expand: nil})
	if err != nil {
		fmt.Printf("Error in getting virtual network detail %v", err)
	}
	fmt.Printf("Response: %v", *vnetRes.Properties.Subnets[0].Properties.RouteTable.Properties.Routes[0])
}

func createRoute(ctx context.Context, clients *Client, subID, rgName, rtbName, rtName string) {
	rt := armnetwork.Route{
		ID:   to.Ptr(fmt.Sprintf(rtIDFormat, subID, rgName, rtbName, rtName)),
		Name: to.Ptr(rtName),
		Properties: &armnetwork.RoutePropertiesFormat{
			NextHopType:      to.Ptr(armnetwork.RouteNextHopTypeVirtualAppliance),
			AddressPrefix:    to.Ptr(defaultAddressPrefix),
			NextHopIPAddress: to.Ptr("10.10.11.2"),
		},
	}
	poller, err := clients.routesClient.BeginCreateOrUpdate(ctx, rgName, rtbName, rtName, rt, &armnetwork.RoutesClientBeginCreateOrUpdateOptions{})

	if err != nil {
		fmt.Printf("Error in polling response to create route %v", err)
		return
	}

	res, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		fmt.Printf("Error in getting response to create route %v", err)
		return
	}
	fmt.Println(res)
}

func createRouteTable(ctx context.Context, clients *Client, rgName, rtbName string) {
	params := armnetwork.RouteTable{
		Tags: map[string]*string{
			"site": to.Ptr("test"),
		},
	}
	poller, err := clients.routeTablesClient.BeginCreateOrUpdate(ctx, rgName, rtbName, params, &armnetwork.RouteTablesClientBeginCreateOrUpdateOptions{})
	if err != nil {
		fmt.Printf("Error in getting route table details %v", err)
		return
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		fmt.Printf("Error in polling response: %v", err)
	}
}
