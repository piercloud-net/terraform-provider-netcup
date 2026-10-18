package provider_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/rixlhq/terraform-provider-netcup/internal/client"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

func TestDNSCredentials(t *testing.T) {
	t.Parallel()
	ccp, err := client.New("12345", "key", "password", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	clients := map[string]*providerdata.Data{
		"SCP only": {SCP: scpclient.New("token", "", "", nil)},
		"CCP":      {CCP: ccp},
		"empty":    {},
	}
	for name, data := range clients {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, newResource := range []func() resource.Resource{provider.NewDNSRecordResource, provider.NewDNSZoneResource} {
				r := newResource().(resource.ResourceWithConfigure)
				var response resource.ConfigureResponse
				r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: data}, &response)
				checkDNSCredentials(t, response.Diagnostics, data.CCP == nil)
			}
			for _, newDataSource := range []func() datasource.DataSource{provider.NewDNSRecordsDataSource, provider.NewDNSZoneDataSource} {
				d := newDataSource().(datasource.DataSourceWithConfigure)
				var response datasource.ConfigureResponse
				d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: data}, &response)
				checkDNSCredentials(t, response.Diagnostics, data.CCP == nil)
			}
		})
	}
}

func checkDNSCredentials(t *testing.T, diagnostics diag.Diagnostics, wantError bool) {
	t.Helper()
	if diagnostics.HasError() != wantError {
		t.Fatalf("configuration diagnostics = %v, want error %v", diagnostics, wantError)
	}
	if wantError && (len(diagnostics) != 1 || diagnostics[0].Summary() != "Missing CCP Credentials") {
		t.Fatalf("expected one missing-credentials error, got %v", diagnostics)
	}
}
