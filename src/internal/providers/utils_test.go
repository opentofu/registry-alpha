package providers

import (
	"testing"
)

func TestFindShaSum(t *testing.T) {
	contents := `0002dd4c79453da5bf1bb9c52172a25d042a571f6df131b7c9ced3d1f8f3eb44  terraform-provider-random_3.5.1_linux_386.zip
49b0f8c2bd5632799aa6113e0e46acaa7d008f927665a41a1f8e8559fe6d8165  terraform-provider-random_3.5.1_darwin_amd64.zip
56df70fca236caa06d0e636c41ab71dd1ced05375f4ddcb905b0ed2105737048  terraform-provider-random_3.5.1_windows_386.zip
58e4de40540c86b9e2e2595dac1318ba057718961a467fa9727866f747693eb2  terraform-provider-random_3.5.1_windows_arm64.zip
5992f11c738812ccd7476d4c607cb8b76dea5aa612be491150c89957ec395ddd  terraform-provider-random_3.5.1_darwin_arm64.zip
7ff4f0b7707b51737f684e96d85a47f0dd8be0f72a3c27b0798755d3faad15e2  terraform-provider-random_3.5.1_linux_arm.zip
8e4b0972e216c9773ab525accfa36eb27c44c751b06b125ecc53f4226c91cea8  terraform-provider-random_3.5.1_linux_arm64.zip
d8956cc5abcd5d1173b6cc25d5d8ed2c5cc456edab2fddb774a17d45e84820cb  terraform-provider-random_3.5.1_linux_amd64.zip
df7f9eb93a832e66bc20cc41c57d38954f87671ec60be09fa866273adb8d9353  terraform-provider-random_3.5.1_windows_amd64.zip
eb583d8f03b11f0b6c535375d8ed0d29e5f7f537b5c78943856d2e8ce76482d9  terraform-provider-random_3.5.1_windows_arm.zip
`

	filename := "terraform-provider-random_3.5.1_linux_amd64.zip"
	shaSum := findShaSum([]byte(contents), filename, "")
	if shaSum != "d8956cc5abcd5d1173b6cc25d5d8ed2c5cc456edab2fddb774a17d45e84820cb" {
		t.Fatal("shaSum not found")
	}
}
