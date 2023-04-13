- should we perform this IBC upgrade? https://github.com/cosmos/ibc-go/blob/v7.0.0/docs/migrations/v6-to-v7.md#chains
- upgrade channelKeeper.SendPacket
- should we use solomachine light client or tm light client ? (currently its
  tm) see https://github.com/cosmos/ibc-go/blob/v7.0.0/docs/migrations/v6-to-v7.md#light-client-registration
- any change related to IBC light client ? https://github.com/cosmos/ibc-go/blob/v7.0.0/docs/migrations/v6-to-v7.md#ibc-light-clients
- replace simapp.MakeTestEncodingConfig by moduletestutil.MakeTestEncodingConfig ?
(see https://github.com/cosmos/cosmos-sdk/blob/release/v0.47.x/UPGRADING.md#encoding)

