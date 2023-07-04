use cosmwasm_schema::cw_serde;
use cosmwasm_std::{Addr, Coin};

#[cw_serde]
pub struct InstantiateMsg {
    pub owner: Addr,
    pub trusted_addresses: Vec<Addr>,
    pub threshold: u32,
}

#[cw_serde]
pub enum ExecuteMsg {
    ThresholdBankSend {
        id: String,
        amount: Coin,
        recipient: Addr,
    },
    Withdraw {},
}

#[cw_serde]
pub enum QueryMsg {
    GetConfig {},
    GetPendingTransaction {
        evidence_id: String,
    },
    GetPendingTransactions {
        offset: Option<u64>,
        limit: Option<u32>,
    },
    GetSentTransaction {
        id: String,
    },
    GetSentTransactions {
        offset: Option<u64>,
        limit: Option<u32>,
    },
}

#[cw_serde]
pub struct Config {
    pub owner: Addr,
    pub trusted_addresses: Vec<Addr>,
    pub threshold: u32,
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
