use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Addr, Coin, Uint128};

#[cw_serde]
pub struct XRPLToken {
    pub currency: String,
    pub issuer: String,
    pub activation_date: u64,
    pub multiplier: String,
}

#[cw_serde]
pub struct BSCToken {
    pub bridge_address: String,
    pub activation_date: u64,
    pub decimals: u8,
}

#[cw_serde]
pub struct InstantiateMsg {
    pub owner: Addr,
    pub trusted_addresses: Vec<Addr>,
    pub threshold: u32,
    pub min_amount: Uint128,
    pub max_amount: Uint128,
    pub xrpl_tokens: Vec<XRPLToken>,
    pub bsc_tokens: Vec<BSCToken>,
}

#[cw_serde]
pub enum ExecuteMsg {
    ThresholdBankSend {
        id: String,
        amount: Coin,
        recipient: Addr,
    },
    ExecutePending {
        evidence_id: String,
    },
    UpdateMinAmount {
        min_amount: Uint128,
    },
    UpdateMaxAmount {
        max_amount: Uint128,
    },
    UpdateTrustedAddresses {
        trusted_addresses: Vec<Addr>,
    },
    AddXrplTokens {
        xrpl_tokens: Vec<XRPLToken>,
    },
    AddBscTokens {
        bsc_tokens: Vec<BSCToken>,
    },
}

#[cw_serde]
pub struct MigrateMsg {}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(ConfigResponse)]
    GetConfig {},
    #[returns(Transaction)]
    GetPendingTransaction { evidence_id: String },
    #[returns(PendingTransactions)]
    GetPendingTransactions {
        offset: Option<u64>,
        limit: Option<u32>,
    },
    #[returns(Transaction)]
    GetSentTransaction { id: String },
    #[returns(SentTransactions)]
    GetSentTransactions {
        offset: Option<u64>,
        limit: Option<u32>,
    },
}

#[cw_serde]
pub struct ConfigResponse {
    pub owner: Addr,
    pub trusted_addresses: Vec<Addr>,
    pub threshold: u32,
    pub min_amount: Uint128,
    pub max_amount: Uint128,
    pub xrpl_tokens: Vec<XRPLToken>,
    pub bsc_tokens: Vec<BSCToken>,
    pub version: u64,
}

#[cw_serde]
pub struct Transaction {
    pub amount: Coin,
    pub recipient: Addr,
    pub evidence_providers: Vec<Addr>,
}

impl Default for Transaction {
    fn default() -> Self {
        Transaction {
            amount: Default::default(),
            recipient: Addr::unchecked(""),
            evidence_providers: vec![],
        }
    }
}

#[cw_serde]
pub struct PendingTransaction {
    pub evidence_id: String,
    pub amount: Coin,
    pub recipient: Addr,
    pub evidence_providers: Vec<Addr>,
}

#[cw_serde]
pub struct SentTransaction {
    pub id: String,
    pub amount: Coin,
    pub recipient: Addr,
    pub evidence_providers: Vec<Addr>,
}

#[cw_serde]
pub struct PendingTransactions {
    pub transactions: Vec<PendingTransaction>,
}

#[cw_serde]
pub struct SentTransactions {
    pub transactions: Vec<SentTransaction>,
}
