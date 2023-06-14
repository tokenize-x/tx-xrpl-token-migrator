use cosmwasm_std::{Addr, Coin};

use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {
    pub owner: Addr,
    pub trusted_addresses: Vec<Addr>,
    pub threshold: u32,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    ThresholdBankSend {
        id: String,
        amount: Coin,
        recipient: Addr,
    },
    Withdraw {},
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
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

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
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

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct PendingTransaction {
    pub evidence_id: String,
    pub amount: Coin,
    pub recipient: Addr,
    pub evidence_providers: Vec<Addr>,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct SentTransaction {
    pub id: String,
    pub amount: Coin,
    pub recipient: Addr,
    pub evidence_providers: Vec<Addr>,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct PendingTransactions {
    pub transactions: Vec<PendingTransaction>,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct SentTransactions {
    pub transactions: Vec<SentTransaction>,
}
