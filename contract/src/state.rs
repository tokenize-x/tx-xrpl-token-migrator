use crate::msg;
use cosmwasm_std::{Addr, Empty, Uint128};
use cw_storage_plus::{Item, Map};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ContractConfig {
    pub owner: Addr,
    pub trusted_addresses: Vec<Addr>,
    pub threshold: u32,
    pub min_amount: Uint128,
    pub max_amount: Uint128,
    pub xrpl_tokens: Vec<msg::XRPLToken>,
    pub version: u64,
}

// Current config
pub const CONFIG: Item<ContractConfig> = Item::new("config");

// Legacy state items for migration
pub const OWNER: Item<Addr> = Item::new("owner");
pub const THRESHOLD: Item<u32> = Item::new("threshold");
pub const TRUSTED_ADDRESSES: Map<Addr, Empty> = Map::new("trusted_addresses");
pub const MIN_AMOUNT: Item<Uint128> = Item::new("min_amount");
pub const MAX_AMOUNT: Item<Uint128> = Item::new("max_amount");

// Transaction storage (unchanged)
pub const PENDING_TRANSACTIONS: Map<String, msg::Transaction> = Map::new("pending_transactions");
pub const SENT_TRANSACTIONS: Map<String, msg::Transaction> = Map::new("sent_transactions");
