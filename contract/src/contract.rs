use cosmwasm_std::{
    entry_point, to_binary, Addr, BankMsg, Binary, Coin, CosmosMsg, Decimal, Deps, DepsMut, Env,
    MessageInfo, Order, Response, StdError, StdResult, Uint128,
};
use cw2::set_contract_version;
use cw_storage_plus::Map;
use cw_utils::one_coin;

use crate::error::ContractError;
use crate::msg::{
    ConfigResponse, ExecuteMsg, InstantiateMsg, MigrateMsg, PendingTransaction,
    PendingTransactions, QueryMsg, SentTransaction, SentTransactions, Transaction,
};
use crate::state::{
    ContractConfig,
    CONFIG,
    MAX_AMOUNT,
    MIN_AMOUNT,
    // Existing state items for migration
    OWNER,
    PENDING_TRANSACTIONS,
    SENT_TRANSACTIONS,
    THRESHOLD,
    TRUSTED_ADDRESSES,
};

const CONTRACT_NAME: &str = env!("CARGO_PKG_NAME");
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

const DEFAULT_PAGE_LIMIT: u32 = 500;
const MAX_PAGE_LIMIT: u32 = DEFAULT_PAGE_LIMIT;

// Migration default values
const DEFAULT_THRESHOLD: u32 = 2;
const DEFAULT_MIN_AMOUNT: Uint128 = Uint128::new(100);
const DEFAULT_MAX_AMOUNT: Uint128 = Uint128::new(200_000_000);

// XRPL validation constants
// Reference: ripple/crypto/const.go
const XRPL_BASE58_ALPHABET: &[u8; 58] =
    b"rpshnaf39wBUDNEGHJKLM4PQRST7VWXYZ2bcdeCg65jkm8oFqi1tuvAxyz";

// XRPL currency constants
// Currency codes are 40-character hexadecimal strings (160 bits)
// Reference: ripple/data/currency.go
const XRPL_CURRENCY_HEX_LENGTH: usize = 40;

// XRPL address validation constants
// Reference: ripple/crypto/const.go
const RIPPLE_ACCOUNT_ID_VERSION: u8 = 0;
const RIPPLE_ACCOUNT_ID_PAYLOAD_LENGTH: usize = 20;
const RIPPLE_ACCOUNT_ID_VERSION_BYTE_LENGTH: usize = 1;
// Decoded address length after checksum is stripped: [version:1][payload:20] = 21 bytes
const RIPPLE_ACCOUNT_ID_DECODED_LENGTH: usize =
    RIPPLE_ACCOUNT_ID_VERSION_BYTE_LENGTH + RIPPLE_ACCOUNT_ID_PAYLOAD_LENGTH;

// Multiplier validation constants
const MIN_MULTIPLIER_NUMERATOR: u128 = 1;
const MIN_MULTIPLIER_DENOMINATOR: u128 = 10; // 0.1 = 1/10
const MAX_MULTIPLIER_NUMERATOR: u128 = 10;
const MAX_MULTIPLIER_DENOMINATOR: u128 = 1; // 10.0 = 10/1

// Token limit constant
const MAX_XRPL_TOKENS: usize = 200;

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
    deps.api.addr_validate(msg.owner.as_str())?;

    if msg.threshold == 0 || msg.threshold > msg.trusted_addresses.len() as u32 {
        return Err(ContractError::InvalidThreshold {});
    }

    // Validate trusted addresses for duplicates
    let mut seen = std::collections::HashSet::new();
    for addr in &msg.trusted_addresses {
        deps.api.addr_validate(addr.as_str())?;
        if !seen.insert(addr.clone()) {
            return Err(ContractError::DuplicatedTrustedAddress {});
        }
    }

    // Validate XRPL tokens if provided
    for token in &msg.xrpl_tokens {
        validate_xrpl_token(token)?;
    }

    // Check for duplicates in initial tokens
    let mut token_seen = std::collections::HashSet::new();
    for token in &msg.xrpl_tokens {
        let key = (token.issuer.clone(), token.currency.clone());
        if !token_seen.insert(key) {
            return Err(ContractError::DuplicatedXRPLToken {});
        }
    }

    // Create initial config
    let config = ContractConfig {
        owner: msg.owner,
        trusted_addresses: msg.trusted_addresses,
        threshold: msg.threshold,
        min_amount: msg.min_amount,
        max_amount: msg.max_amount,
        xrpl_tokens: msg.xrpl_tokens,
        version: 1,
    };

    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::ThresholdBankSend {
            id,
            amount,
            recipient,
        } => threshold_bank_send(deps, info, id, amount, recipient),
        ExecuteMsg::ExecutePending { evidence_id } => execute_pending(deps, info, evidence_id),
        ExecuteMsg::UpdateMinAmount { min_amount } => update_min_amount(deps, info, min_amount),
        ExecuteMsg::UpdateMaxAmount { max_amount } => update_max_amount(deps, info, max_amount),
        ExecuteMsg::UpdateTrustedAddresses { trusted_addresses } => {
            update_trusted_addresses(deps, info, trusted_addresses)
        }
        ExecuteMsg::AddXrplTokens { xrpl_tokens } => add_xrpl_tokens(deps, info, xrpl_tokens),
    }
}

#[entry_point]
pub fn migrate(deps: DepsMut, _env: Env, _msg: MigrateMsg) -> Result<Response, ContractError> {
    let storage_version = cw2::get_contract_version(deps.storage)?;
    if storage_version.contract != CONTRACT_NAME {
        return Err(StdError::generic_err("Can only upgrade from same contract name").into());
    }

    // Check if migration is needed (config doesn't exist yet)
    if CONFIG.may_load(deps.storage)?.is_none() {
        // Migration from old state structure

        // Read existing state items
        let owner = OWNER.may_load(deps.storage)?;
        let threshold = THRESHOLD.may_load(deps.storage)?;
        let min_amount = MIN_AMOUNT.may_load(deps.storage)?;
        let max_amount = MAX_AMOUNT.may_load(deps.storage)?;

        // Collect trusted addresses from existing map
        let trusted_addresses: Vec<Addr> = TRUSTED_ADDRESSES
            .keys(deps.storage, None, None, Order::Ascending)
            .filter_map(|r| r.ok())
            .collect();

        // Use defaults for missing values
        let owner = owner.unwrap_or_else(|| _env.contract.address.clone());
        let mut threshold = threshold.unwrap_or(DEFAULT_THRESHOLD);
        let min_amount = min_amount.unwrap_or(DEFAULT_MIN_AMOUNT);
        let max_amount = max_amount.unwrap_or(DEFAULT_MAX_AMOUNT);

        // Ensure threshold is valid (must be > 0 and <= trusted_addresses.len())
        // If threshold is invalid, use appropriate default
        if threshold == 0
            || (!trusted_addresses.is_empty() && threshold > trusted_addresses.len() as u32)
        {
            // If no trusted addresses, default to 1; otherwise use DEFAULT_THRESHOLD
            threshold = if trusted_addresses.is_empty() {
                1
            } else {
                // Ensure default doesn't exceed trusted_addresses length
                DEFAULT_THRESHOLD.min(trusted_addresses.len() as u32)
            };
        }

        // Create config from existing state with defaults applied
        // xrpl_tokens starts empty - will be set via add_xrpl_tokens
        let config = ContractConfig {
            owner,
            trusted_addresses,
            threshold,
            min_amount,
            max_amount,
            xrpl_tokens: vec![],
            version: 1,
        };

        // Save config
        CONFIG.save(deps.storage, &config)?;

        // Note: Existing state items are NOT removed here for safety
        // They will be removed in a future contract version after migration is verified
    }

    // Upgrade contract version
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;

    Ok(Response::default())
}

pub fn threshold_bank_send(
    deps: DepsMut,
    info: MessageInfo,
    id: String,
    amount: Coin,
    recipient: Addr,
) -> Result<Response, ContractError> {
    let id = normalize_id(id);
    deps.api.addr_validate(recipient.as_str())?;

    let config = CONFIG.load(deps.storage)?;

    if !config.trusted_addresses.contains(&info.sender) {
        return Err(ContractError::Unauthorized {});
    }

    if config.min_amount.gt(&amount.amount) {
        return Err(ContractError::LowAmount {});
    }

    // This check prevents sending the transaction with the same ID.
    // Once the number of the trusted addresses by the evidence ID reaches the threshold
    // the contract executes the bank send transaction and prohibits the execution of the transaction
    // with the same ID again.
    if SENT_TRANSACTIONS.has(deps.storage, id.clone()) {
        return Err(ContractError::TransferAlreadySent {});
    }

    // The evidence ID is: `id-amountdenom-recipient` to cover the case when some of the
    // trusted addresses send the message with the same id but different amount or denom or recipient,
    // in that case the transaction will be added to pending queue, but executed only once/if it
    // reaches the threshold
    let mut tx: Transaction;
    let evidence_id = build_evidence_id(id.clone(), amount.clone(), recipient.clone());
    match PENDING_TRANSACTIONS.may_load(deps.storage, evidence_id.clone())? {
        None => {
            tx = Transaction {
                recipient,
                amount: amount.clone(),
                evidence_providers: vec![info.sender],
            };
        }
        Some(stored_tx) => {
            tx = stored_tx;
            for evidence in tx.evidence_providers.clone() {
                if evidence == info.sender.clone() {
                    return Err(ContractError::EvidenceAlreadyProvided {});
                }
            }
            tx.evidence_providers.push(info.sender)
        }
    }

    // execute transaction if it doesn't exceed max amount
    if tx.evidence_providers.len() as u32 == config.threshold
        && config.max_amount.ge(&amount.amount.clone())
    {
        return Ok(send_bank_transaction(deps, &tx, id, evidence_id)?);
    }

    PENDING_TRANSACTIONS.save(deps.storage, evidence_id.clone(), &tx)?;
    Ok(Response::new()
        .add_attribute("result", "pending")
        .add_attribute("evidence_id", evidence_id))
}

pub fn execute_pending(
    deps: DepsMut,
    info: MessageInfo,
    evidence_id: String,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;

    match PENDING_TRANSACTIONS.may_load(deps.storage, evidence_id.clone())? {
        None => Err(ContractError::TransactionNotFound {}),
        Some(tx) => {
            if (tx.evidence_providers.len() as u32) < config.threshold {
                return Err(ContractError::TransactionNotConfirmed {});
            }
            let funds_sent = one_coin(&info)?;
            // check that sender covers the amount in the pending transaction
            if !funds_sent.eq(&tx.amount) {
                return Err(ContractError::FundsMismatch {});
            }
            Ok(send_bank_transaction(
                deps,
                &tx,
                extract_id_from_evidence_id(evidence_id.clone()),
                evidence_id,
            )?)
        }
    }
}

fn send_bank_transaction(
    deps: DepsMut,
    tx: &Transaction,
    id: String,
    evidence_id: String,
) -> StdResult<Response> {
    let bank_send_msg: CosmosMsg = BankMsg::Send {
        to_address: tx.recipient.clone().into(),
        amount: vec![tx.amount.clone()],
    }
    .into();
    SENT_TRANSACTIONS.save(deps.storage, id, tx)?;
    PENDING_TRANSACTIONS.remove(deps.storage, evidence_id.clone());
    Ok(Response::new()
        .add_attribute("result", "sent")
        .add_attribute("recipient", tx.recipient.to_string())
        .add_attribute("amount", tx.amount.to_string())
        .add_attribute("evidence_id", evidence_id)
        .add_message(bank_send_msg))
}

pub fn update_min_amount(
    deps: DepsMut,
    info: MessageInfo,
    min_amount: Uint128,
) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;

    if info.sender != config.owner {
        return Err(ContractError::Unauthorized {});
    }

    config.min_amount = min_amount;
    config.version += 1;
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new()
        .add_attribute("action", "update_min_amount")
        .add_attribute("min_amount", min_amount)
        .add_attribute("version", config.version.to_string()))
}

pub fn update_max_amount(
    deps: DepsMut,
    info: MessageInfo,
    max_amount: Uint128,
) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;

    if info.sender != config.owner {
        return Err(ContractError::Unauthorized {});
    }

    config.max_amount = max_amount;
    config.version += 1;
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new()
        .add_attribute("action", "update_max_amount")
        .add_attribute("max_amount", max_amount)
        .add_attribute("version", config.version.to_string()))
}

pub fn update_trusted_addresses(
    deps: DepsMut,
    info: MessageInfo,
    trusted_addresses: Vec<Addr>,
) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;

    if info.sender != config.owner {
        return Err(ContractError::Unauthorized {});
    }

    // Validate and check for duplicates
    let mut seen = std::collections::HashSet::new();
    for addr in &trusted_addresses {
        deps.api.addr_validate(addr.as_str())?;
        if !seen.insert(addr.clone()) {
            return Err(ContractError::DuplicatedTrustedAddress {});
        }
    }

    config.trusted_addresses = trusted_addresses;
    config.version += 1;
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new()
        .add_attribute("action", "update_trusted_addresses")
        .add_attribute("version", config.version.to_string()))
}

pub fn add_xrpl_tokens(
    deps: DepsMut,
    info: MessageInfo,
    new_tokens: Vec<crate::msg::XRPLToken>,
) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;

    if info.sender != config.owner {
        return Err(ContractError::Unauthorized {});
    }

    // Build unique keys from existing tokens to prevent duplicates
    let mut seen_keys: std::collections::HashSet<(String, String)> = config
        .xrpl_tokens
        .iter()
        .map(|t| (t.issuer.clone(), t.currency.clone()))
        .collect();

    for token in new_tokens {
        // Run our new optimized validation
        validate_xrpl_token(&token)?;

        let key = (token.issuer.clone(), token.currency.clone());
        if !seen_keys.insert(key) {
            return Err(ContractError::DuplicatedXRPLToken {});
        }

        config.xrpl_tokens.push(token);
    }

    // Security Cap: Prevent vector bloat
    // TODO: Discuss with team if we need this limit or alternatively store it in state separately.
    // not in the config struct.
    if config.xrpl_tokens.len() > MAX_XRPL_TOKENS {
        return Err(StdError::generic_err("Maximum token limit reached").into());
    }

    config.version += 1;
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new()
        .add_attribute("action", "add_xrpl_tokens")
        .add_attribute("version", config.version.to_string())
        .add_attribute("total_tokens", config.xrpl_tokens.len().to_string()))
}

fn validate_xrpl_token(token: &crate::msg::XRPLToken) -> Result<(), ContractError> {
    // Validate Currency: 40-char Hex
    if token.currency.len() != XRPL_CURRENCY_HEX_LENGTH
        || !token.currency.chars().all(|c| c.is_ascii_hexdigit())
    {
        return Err(ContractError::InvalidXRPLCurrency {
            reason: format!(
                "Currency must be a {}-character hexadecimal string",
                XRPL_CURRENCY_HEX_LENGTH
            ),
        });
    }

    // Validate Issuer: Use bs58 with built-in checksum (double-sha256) validation
    let decoded = bs58::decode(&token.issuer)
        .with_alphabet(&bs58::Alphabet::new(XRPL_BASE58_ALPHABET).unwrap())
        .with_check(None) // Automatically verifies the 4-byte checksum
        .into_vec()
        .map_err(|_| ContractError::InvalidXRPLIssuer {
            reason: "Invalid Base58 format or checksum mismatch".into(),
        })?;

    // Validate structure: [Version Byte (0x00)][AccountID (20 bytes)]
    if decoded.len() != RIPPLE_ACCOUNT_ID_DECODED_LENGTH || decoded[0] != RIPPLE_ACCOUNT_ID_VERSION
    {
        return Err(ContractError::InvalidXRPLIssuer {
            reason: "Invalid XRPL address version or payload length".into(),
        });
    }

    // Validate Multiplier: Use Decimal for deterministic math
    let val: Decimal = token
        .multiplier
        .parse()
        .map_err(|_| ContractError::InvalidMultiplier {
            reason: format!("Multiplier '{}' is not a valid decimal", token.multiplier),
        })?;

    let min_bound = Decimal::from_ratio(MIN_MULTIPLIER_NUMERATOR, MIN_MULTIPLIER_DENOMINATOR);
    let max_bound = Decimal::from_ratio(MAX_MULTIPLIER_NUMERATOR, MAX_MULTIPLIER_DENOMINATOR);

    if val < min_bound || val > max_bound {
        return Err(ContractError::InvalidMultiplier {
            reason: format!("Multiplier {} is out of allowed range [0.1, 10.0]", val),
        });
    }

    Ok(())
}

fn normalize_id(id: String) -> String {
    // use lowercase for hex ids.
    id.to_lowercase()
}

fn build_evidence_id(id: String, amount: Coin, recipient: Addr) -> String {
    format!("{id}-{amount}-{recipient}").to_lowercase()
}

fn extract_id_from_evidence_id(evidence_id: String) -> String {
    let evidence = evidence_id.split('-').next();
    evidence.unwrap_or_default().to_string()
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetConfig {} => to_binary(&get_config(deps)?),
        QueryMsg::GetPendingTransaction { evidence_id } => {
            to_binary(&get_pending_transaction(deps, evidence_id)?)
        }
        QueryMsg::GetPendingTransactions { offset, limit } => {
            to_binary(&get_pending_transactions(deps, offset, limit)?)
        }
        QueryMsg::GetSentTransaction { id } => to_binary(&get_sent_transaction(deps, id)?),
        QueryMsg::GetSentTransactions { offset, limit } => {
            to_binary(&get_sent_transactions(deps, offset, limit)?)
        }
    }
}

fn get_config(deps: Deps) -> StdResult<ConfigResponse> {
    let config = CONFIG.load(deps.storage)?;

    Ok(ConfigResponse {
        owner: config.owner,
        trusted_addresses: config.trusted_addresses,
        threshold: config.threshold,
        min_amount: config.min_amount,
        max_amount: config.max_amount,
        xrpl_tokens: config.xrpl_tokens,
        version: config.version,
    })
}

fn get_pending_transaction(deps: Deps, evidence_id: String) -> StdResult<Transaction> {
    Ok(PENDING_TRANSACTIONS
        .may_load(deps.storage, evidence_id)?
        .unwrap_or_default())
}

fn get_pending_transactions(
    deps: Deps,
    offset: Option<u64>,
    limit: Option<u32>,
) -> StdResult<PendingTransactions> {
    let transactions = paginate_transactions(
        deps,
        PENDING_TRANSACTIONS,
        offset,
        limit,
        |k: String, v: Transaction| PendingTransaction {
            evidence_id: k,
            amount: v.amount,
            recipient: v.recipient,
            evidence_providers: v.evidence_providers,
        },
    );
    Ok(PendingTransactions { transactions })
}

fn get_sent_transaction(deps: Deps, id: String) -> StdResult<Transaction> {
    let id = normalize_id(id);
    Ok(SENT_TRANSACTIONS
        .may_load(deps.storage, id)?
        .unwrap_or_default())
}

fn get_sent_transactions(
    deps: Deps,
    offset: Option<u64>,
    limit: Option<u32>,
) -> StdResult<SentTransactions> {
    let transactions = paginate_transactions(
        deps,
        SENT_TRANSACTIONS,
        offset,
        limit,
        |k: String, v: Transaction| SentTransaction {
            id: k,
            amount: v.amount,
            recipient: v.recipient,
            evidence_providers: v.evidence_providers,
        },
    );
    Ok(SentTransactions { transactions })
}

fn paginate_transactions<T>(
    deps: Deps,
    map: Map<String, Transaction>,
    offset: Option<u64>,
    limit: Option<u32>,
    mapper: impl Fn(String, Transaction) -> T,
) -> Vec<T> {
    let limit = limit.unwrap_or(DEFAULT_PAGE_LIMIT).min(MAX_PAGE_LIMIT) as usize;
    let offset = offset.unwrap_or(0) as usize;
    map.range(deps.storage, None, None, Order::Ascending)
        .skip(offset)
        .take(limit)
        .filter_map(|v| v.ok())
        .map(|(k, v)| mapper(k, v))
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{mock_dependencies, mock_env, mock_info};
    use cosmwasm_std::{coin, StdError, Uint128};
    use cw_utils::PaymentError;
    use std::ops::{Add, Sub};

    const TEST_OWNER: &str = "devcore19ptkyamzervlx3rk39n08x5mtqxsuak8fk0try";
    const TEST_TRUSTED_ADDRESS1: &str = "devcore1h6ad6g6tpfwajjd4xmuu2pqvsxunmnxwxzfrc2";
    const TEST_TRUSTED_ADDRESS2: &str = "devcore1wvkmnjken95y3aesnmre052u0f0azdl8y499th";
    const TEST_TRUSTED_ADDRESS3: &str = "devcore1zwhqfg9m9j20cy73mcg64hfkdx2gudzhhy0t6y";
    const TEST_ANY_ADDRESS: &str = "devcore1csqfjmzjevslxcve3aaynr0wfskyhp75qmmydh";
    const TEST_THRESHOLD: u32 = 2;
    const TEST_MIN_AMOUNT: Uint128 = Uint128::new(10);
    const TEST_MAX_AMOUNT: Uint128 = Uint128::new(1000);

    // XRPL token test constants
    const TEST_XRPL_CORE_CURRENCY: &str = "434F524500000000000000000000000000000000";
    const TEST_XRPL_CORE_ISSUER: &str = "raSEP47QAwU6jsZU493znUD2iGNHDQEyvA";
    const TEST_XRPL_XCORE_CURRENCY: &str = "58434F5245000000000000000000000000000000";
    const TEST_XRPL_XCORE_ISSUER: &str = "rawnyFwFLkntQttzBgEFiASg5iB5ULdKpX";
    const TEST_XRPL_SOLO_CURRENCY: &str = "534F4C4F00000000000000000000000000000000";
    const TEST_XRPL_SOLO_ISSUER: &str = "rHZwvHEs56GCmHupwjA4RY7oPA3EoAJWuN";
    const TEST_ACTIVATION_DATE: u64 = 946684800; // time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
    const TEST_MULTIPLIER_1_0: &str = "1.0";
    const TEST_MULTIPLIER_1_25: &str = "1.25";

    fn init_msg() -> InstantiateMsg {
        InstantiateMsg {
            owner: Addr::unchecked(TEST_OWNER),
            threshold: TEST_THRESHOLD,
            trusted_addresses: vec![
                Addr::unchecked(TEST_TRUSTED_ADDRESS1),
                Addr::unchecked(TEST_TRUSTED_ADDRESS2),
                Addr::unchecked(TEST_TRUSTED_ADDRESS3),
            ],
            min_amount: TEST_MIN_AMOUNT,
            max_amount: TEST_MAX_AMOUNT,
            xrpl_tokens: vec![],
        }
    }

    #[test]
    fn test_instantiate() {
        let test_cases = &[
            ("Valid instantiate", { init_msg() }, "".to_string()),
            (
                "Invalid owner address",
                {
                    let mut msg = init_msg();
                    msg.owner = Addr::unchecked("INVALID");
                    msg
                },
                StdError::generic_err("Invalid input: address not normalized").to_string(),
            ),
            (
                "Threshold is zero",
                {
                    let mut msg = init_msg();
                    msg.threshold = 0;
                    msg
                },
                ContractError::InvalidThreshold {}.to_string(),
            ),
            (
                "Threshold is greater than the addresses len",
                {
                    let mut msg = init_msg();
                    msg.threshold = 4;
                    msg
                },
                ContractError::InvalidThreshold {}.to_string(),
            ),
            (
                "Duplicated trusted address",
                {
                    let mut msg = init_msg();
                    msg.trusted_addresses = vec![
                        Addr::unchecked(TEST_TRUSTED_ADDRESS1),
                        Addr::unchecked(TEST_TRUSTED_ADDRESS1),
                        Addr::unchecked(TEST_TRUSTED_ADDRESS2),
                    ];
                    msg
                },
                ContractError::DuplicatedTrustedAddress {}.to_string(),
            ),
        ];

        for (name, msg, expected_err) in test_cases {
            println!("Test case: {}", name);
            let mut deps = mock_dependencies();
            let env = mock_env();
            let info = mock_info(TEST_OWNER, &[]);

            let res = instantiate(deps.as_mut(), env.clone(), info.clone(), msg.clone());
            if !expected_err.is_empty() {
                assert_eq!(expected_err.to_string(), res.err().unwrap().to_string());
                continue;
            }

            let config = get_config(deps.as_ref()).unwrap();

            assert_eq!(TEST_OWNER, config.owner);
            assert_eq!(TEST_THRESHOLD, config.threshold);
            assert_eq!(
                vec![
                    TEST_TRUSTED_ADDRESS1.to_string(),
                    TEST_TRUSTED_ADDRESS2.to_string(),
                    TEST_TRUSTED_ADDRESS3.to_string(),
                ],
                config.trusted_addresses
            );
            assert_eq!(TEST_MIN_AMOUNT, config.min_amount);
            assert_eq!(TEST_MAX_AMOUNT, config.max_amount);
        }
    }

    #[test]
    fn test_threshold_bank_send() {
        let mut deps = mock_dependencies();
        let env = mock_env();
        let info = mock_info(TEST_OWNER, &[]);

        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), init_msg());
        assert!(res.is_ok());

        let id: String = "tx_hash1".to_string();
        let denom: String = "ucore".to_string();
        let low_amount: Coin = coin(9_u128, denom.clone());
        let amount: Coin = coin(999_u128, denom.clone());
        let malicious_amount: Coin = coin(777_u128, denom.clone());

        let recipient: Addr = Addr::unchecked("devcore1y9cnpjxwa7xc5nuhvzzsu23d04jfc6vkrgx4k5");
        let evidence_id = build_evidence_id(id.clone(), amount.clone(), recipient.clone());
        let malicious_evidence_id =
            build_evidence_id(id.clone(), malicious_amount.clone(), recipient.clone());

        // sending to invalid recipient address
        let info = mock_info(TEST_TRUSTED_ADDRESS1, &[]);
        let res = threshold_bank_send(
            deps.as_mut(),
            info.clone(),
            id.clone(),
            amount.clone(),
            Addr::unchecked("INVALID"),
        );
        assert_eq!(
            StdError::generic_err("Invalid input: address not normalized").to_string(),
            res.err().unwrap().to_string()
        );

        // sending from not trusted address
        let info = mock_info(TEST_OWNER, &[]);
        let res = threshold_bank_send(
            deps.as_mut(),
            info.clone(),
            id.clone(),
            amount.clone(),
            recipient.clone(),
        );
        assert_eq!(ContractError::Unauthorized {}, res.err().unwrap());

        // sending low amount
        let info = mock_info(TEST_TRUSTED_ADDRESS1, &[]);
        let res = threshold_bank_send(
            deps.as_mut(),
            info.clone(),
            id.clone(),
            low_amount.clone(),
            recipient.clone(),
        );
        assert_eq!(ContractError::LowAmount {}, res.err().unwrap());

        // sending from the first valid trusted address
        let info = mock_info(TEST_TRUSTED_ADDRESS1, &[]);
        let res = threshold_bank_send(
            deps.as_mut(),
            info.clone(),
            id.clone(),
            amount.clone(),
            recipient.clone(),
        );
        // check that no messages were produced
        assert_eq!(0, res.unwrap().messages.len());
        // check that tx is pending now
        let pending_tx = get_pending_transaction(deps.as_ref(), evidence_id.clone()).unwrap();
        assert_eq!(
            Transaction {
                amount: amount.clone(),
                recipient: recipient.clone(),
                evidence_providers: vec![Addr::unchecked(TEST_TRUSTED_ADDRESS1)],
            },
            pending_tx
        );

        // try to send same tx with the same trusted address
        let info = mock_info(TEST_TRUSTED_ADDRESS1, &[]);
        let res = threshold_bank_send(
            deps.as_mut(),
            info.clone(),
            id.clone(),
            amount.clone(),
            recipient.clone(),
        );
        assert_eq!(
            ContractError::EvidenceAlreadyProvided {},
            res.err().unwrap()
        );

        // try to send tx with the same id but different data from the second trusted address
        let info = mock_info(TEST_TRUSTED_ADDRESS2, &[]);
        let res = threshold_bank_send(
            deps.as_mut(),
            info.clone(),
            id.clone(),
            malicious_amount.clone(),
            recipient.clone(),
        );
        // check that no messages were produced
        assert_eq!(0, res.unwrap().messages.len());
        // check that tx is pending now
        let pending_tx =
            get_pending_transaction(deps.as_ref(), malicious_evidence_id.clone()).unwrap();
        assert_eq!(
            Transaction {
                amount: malicious_amount.clone(),
                recipient: recipient.clone(),
                evidence_providers: vec![Addr::unchecked(TEST_TRUSTED_ADDRESS2)],
            },
            pending_tx
        );

        // sending from the third valid trusted address
        let info = mock_info(TEST_TRUSTED_ADDRESS3, &[]);
        let res = threshold_bank_send(
            deps.as_mut(),
            info.clone(),
            id.clone(),
            amount.clone(),
            recipient.clone(),
        );
        // check that bank send was submitted
        let res_msgs = res.unwrap().messages;
        assert_eq!(1, res_msgs.len());
        let bank_send_msg: CosmosMsg = BankMsg::Send {
            to_address: recipient.clone().into(),
            amount: vec![amount.clone()],
        }
        .into();
        assert_eq!(bank_send_msg, res_msgs[0].msg);
        // check that tx not pending now
        let pending_tx = get_pending_transaction(deps.as_ref(), evidence_id).unwrap();
        assert_eq!(
            Transaction {
                amount: coin(Uint128::zero().u128(), ""),
                recipient: Addr::unchecked(""),
                evidence_providers: vec![],
            },
            pending_tx
        );

        // check that tx is sent
        let sent_tx = get_sent_transaction(deps.as_ref(), id.clone()).unwrap();
        assert_eq!(
            Transaction {
                amount: amount.clone(),
                recipient: recipient.clone(),
                evidence_providers: vec![
                    Addr::unchecked(TEST_TRUSTED_ADDRESS1),
                    Addr::unchecked(TEST_TRUSTED_ADDRESS3),
                ],
            },
            sent_tx
        );

        // try to send the tx with same id(hash) from the valid trusted address
        let info = mock_info(TEST_TRUSTED_ADDRESS3, &[]);
        let res = threshold_bank_send(
            deps.as_mut(),
            info.clone(),
            id.clone(),
            amount.clone(),
            recipient.clone(),
        );
        assert_eq!(ContractError::TransferAlreadySent {}, res.err().unwrap());

        // try to send the tx with the evidence ID which is still ending,
        // and with id(hash) which is already sent
        let info = mock_info(TEST_TRUSTED_ADDRESS3, &[]);
        let res = threshold_bank_send(
            deps.as_mut(),
            info.clone(),
            id.clone(),
            malicious_amount.clone(),
            recipient.clone(),
        );
        assert_eq!(ContractError::TransferAlreadySent {}, res.err().unwrap());
    }

    #[test]
    fn test_execute_pending() {
        let mut deps = mock_dependencies();
        let env = mock_env();
        let info = mock_info(TEST_OWNER, &[]);

        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), init_msg());
        assert!(res.is_ok());

        let id: String = "tx_hash1".to_string();
        let denom: String = "ucore".to_string();
        let amount: Coin = coin(20_000_u128, denom.clone());

        let recipient: Addr = Addr::unchecked("devcore1y9cnpjxwa7xc5nuhvzzsu23d04jfc6vkrgx4k5");
        let evidence_id = build_evidence_id(id.clone(), amount.clone(), recipient.clone());

        // try to execute with invalid evidence id
        let info = mock_info(TEST_ANY_ADDRESS, &[]);
        let res = execute_pending(deps.as_mut(), info.clone(), "invalid".to_string());
        assert_eq!(ContractError::TransactionNotFound {}, res.err().unwrap());

        // sending from the first trusted address
        let info = mock_info(TEST_TRUSTED_ADDRESS1, &[]);
        let res = threshold_bank_send(
            deps.as_mut(),
            info.clone(),
            id.clone(),
            amount.clone(),
            recipient.clone(),
        );
        // check that no messages were produced
        assert_eq!(0, res.unwrap().messages.len());
        // check that tx is pending now
        let pending_tx = get_pending_transaction(deps.as_ref(), evidence_id.clone()).unwrap();
        assert_eq!(
            Transaction {
                amount: amount.clone(),
                recipient: recipient.clone(),
                evidence_providers: vec![Addr::unchecked(TEST_TRUSTED_ADDRESS1)],
            },
            pending_tx
        );

        // try to execute not confirmed
        let info = mock_info(TEST_ANY_ADDRESS, &[]);
        let res = execute_pending(deps.as_mut(), info.clone(), evidence_id.clone());
        assert_eq!(
            ContractError::TransactionNotConfirmed {},
            res.err().unwrap()
        );

        // sending from the second trusted address
        let info = mock_info(TEST_TRUSTED_ADDRESS2, &[]);
        let res = threshold_bank_send(
            deps.as_mut(),
            info.clone(),
            id.clone(),
            amount.clone(),
            recipient.clone(),
        );
        // check that no messages were produced
        assert_eq!(0, res.unwrap().messages.len());
        // check that tx is still pending
        let pending_tx = get_pending_transaction(deps.as_ref(), evidence_id.clone()).unwrap();
        assert_eq!(
            Transaction {
                amount: amount.clone(),
                recipient: recipient.clone(),
                evidence_providers: vec![
                    Addr::unchecked(TEST_TRUSTED_ADDRESS1),
                    Addr::unchecked(TEST_TRUSTED_ADDRESS2),
                ],
            },
            pending_tx
        );

        // try to execute without funds
        let info = mock_info(TEST_ANY_ADDRESS, &[]);
        let res = execute_pending(deps.as_mut(), info.clone(), evidence_id.clone());
        assert_eq!(
            ContractError::Payment(PaymentError::NoFunds {}),
            res.err().unwrap()
        );

        // try to execute with less amount
        let info = mock_info(
            TEST_ANY_ADDRESS,
            &[coin(
                u128::from(amount.amount.sub(Uint128::new(1))),
                denom.clone(),
            )],
        );
        let res = execute_pending(deps.as_mut(), info.clone(), evidence_id.clone());
        assert_eq!(ContractError::FundsMismatch {}, res.err().unwrap());

        // try to execute with higher amount
        let info = mock_info(
            TEST_ANY_ADDRESS,
            &[coin(
                u128::from(amount.amount.add(Uint128::new(1))),
                denom.clone(),
            )],
        );
        let res = execute_pending(deps.as_mut(), info.clone(), evidence_id.clone());
        assert_eq!(ContractError::FundsMismatch {}, res.err().unwrap());

        // execute with correct amount
        let info = mock_info(TEST_ANY_ADDRESS, &[amount.clone()]);
        let res = execute_pending(deps.as_mut(), info.clone(), evidence_id.clone());

        // check that bank send was submitted
        let res_msgs = res.unwrap().messages;
        assert_eq!(1, res_msgs.len());
        let bank_send_msg: CosmosMsg = BankMsg::Send {
            to_address: recipient.clone().into(),
            amount: vec![amount.clone()],
        }
        .into();
        assert_eq!(bank_send_msg, res_msgs[0].msg);
        // check that tx isn't pending now
        let pending_tx = get_pending_transaction(deps.as_ref(), evidence_id).unwrap();
        assert_eq!(
            Transaction {
                amount: coin(Uint128::zero().u128(), ""),
                recipient: Addr::unchecked(""),
                evidence_providers: vec![],
            },
            pending_tx
        );

        // check that tx is sent
        let sent_tx = get_sent_transaction(deps.as_ref(), id.clone()).unwrap();
        assert_eq!(
            Transaction {
                amount: amount.clone(),
                recipient: recipient.clone(),
                evidence_providers: vec![
                    Addr::unchecked(TEST_TRUSTED_ADDRESS1),
                    Addr::unchecked(TEST_TRUSTED_ADDRESS2),
                ],
            },
            sent_tx
        );
    }

    #[test]
    fn test_update_min_max_amount() {
        let mut deps = mock_dependencies();
        let env = mock_env();

        let info = mock_info(TEST_OWNER, &[]);
        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), init_msg());
        assert!(res.is_ok());

        let config = get_config(deps.as_ref()).unwrap();
        assert_eq!(TEST_MIN_AMOUNT, config.min_amount);
        assert_eq!(TEST_MAX_AMOUNT, config.max_amount);

        let new_min_amount = Uint128::new(123);
        // execute from non-owner
        let info = mock_info(TEST_ANY_ADDRESS, &[]);
        let res = update_min_amount(deps.as_mut(), info.clone(), new_min_amount);
        assert_eq!(ContractError::Unauthorized {}, res.unwrap_err());
        // execute from owner
        let info = mock_info(TEST_OWNER, &[]);
        let res = update_min_amount(deps.as_mut(), info.clone(), new_min_amount);
        assert!(res.is_ok());
        let config = get_config(deps.as_ref()).unwrap();
        assert_eq!(new_min_amount, config.min_amount);

        let new_max_amount = Uint128::new(321);
        // execute from non-owner
        let info = mock_info(TEST_ANY_ADDRESS, &[]);
        let res = update_max_amount(deps.as_mut(), info.clone(), new_max_amount);
        assert_eq!(ContractError::Unauthorized {}, res.unwrap_err());
        // execute from owner
        let info = mock_info(TEST_OWNER, &[]);
        let res = update_max_amount(deps.as_mut(), info, new_max_amount);
        assert!(res.is_ok());
        let config = get_config(deps.as_ref()).unwrap();
        assert_eq!(new_max_amount, config.max_amount);
    }

    #[test]
    fn test_update_trusted_addresses() {
        let mut deps = mock_dependencies();
        let env = mock_env();

        let info = mock_info(TEST_OWNER, &[]);
        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), init_msg());
        assert!(res.is_ok());

        let new_trusted_addresses = vec![
            Addr::unchecked(TEST_TRUSTED_ADDRESS1),
            Addr::unchecked(TEST_TRUSTED_ADDRESS3),
        ];
        // execute from non-owner
        let info = mock_info(TEST_ANY_ADDRESS, &[]);
        let res =
            update_trusted_addresses(deps.as_mut(), info.clone(), new_trusted_addresses.clone());
        assert_eq!(ContractError::Unauthorized {}, res.unwrap_err());
        // execute from owner
        let info = mock_info(TEST_OWNER, &[]);
        let res =
            update_trusted_addresses(deps.as_mut(), info.clone(), new_trusted_addresses.clone());
        assert!(res.is_ok());
        let config = get_config(deps.as_ref()).unwrap();
        assert_eq!(new_trusted_addresses, config.trusted_addresses);
    }

    #[test]
    fn test_xrpl_tokens_in_config() {
        let mut deps = mock_dependencies();
        let env = mock_env();

        let info = mock_info(TEST_OWNER, &[]);
        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), init_msg());
        assert!(res.is_ok());

        // check that config includes xrpl_tokens
        let config = get_config(deps.as_ref()).unwrap();
        assert_eq!(vec![] as Vec<crate::msg::XRPLToken>, config.xrpl_tokens);
    }

    #[test]
    fn test_add_xrpl_tokens() {
        let mut deps = mock_dependencies();
        let env = mock_env();

        let info = mock_info(TEST_OWNER, &[]);
        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), init_msg());
        assert!(res.is_ok());

        let first_xrpl_tokens = vec![
            crate::msg::XRPLToken {
                currency: TEST_XRPL_CORE_CURRENCY.to_string(),
                issuer: TEST_XRPL_CORE_ISSUER.to_string(),
                activation_date: TEST_ACTIVATION_DATE,
                multiplier: TEST_MULTIPLIER_1_0.to_string(),
            },
            crate::msg::XRPLToken {
                currency: TEST_XRPL_XCORE_CURRENCY.to_string(),
                issuer: TEST_XRPL_XCORE_ISSUER.to_string(),
                activation_date: TEST_ACTIVATION_DATE,
                multiplier: TEST_MULTIPLIER_1_0.to_string(),
            },
        ];

        // execute from non-owner should fail
        let info = mock_info(TEST_ANY_ADDRESS, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), first_xrpl_tokens.clone());
        assert_eq!(ContractError::Unauthorized {}, res.unwrap_err());

        // execute from owner should succeed
        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), first_xrpl_tokens.clone());
        assert!(res.is_ok());

        // verify tokens were added
        let config = get_config(deps.as_ref()).unwrap();
        assert_eq!(first_xrpl_tokens, config.xrpl_tokens);

        // Add more tokens - should append, not replace
        let second_xrpl_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_SOLO_CURRENCY.to_string(),
            issuer: TEST_XRPL_SOLO_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: TEST_MULTIPLIER_1_25.to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), second_xrpl_tokens.clone());
        assert!(res.is_ok());

        // verify tokens were appended (immutability - existing tokens remain)
        let config = get_config(deps.as_ref()).unwrap();
        let mut expected_tokens = first_xrpl_tokens.clone();
        expected_tokens.extend(second_xrpl_tokens);
        assert_eq!(expected_tokens, config.xrpl_tokens);

        // Try to add duplicate token - should fail
        let duplicate_token = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_CORE_CURRENCY.to_string(),
            issuer: TEST_XRPL_CORE_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: TEST_MULTIPLIER_1_0.to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), duplicate_token);
        assert_eq!(ContractError::DuplicatedXRPLToken {}, res.unwrap_err());

        // Verify tokens remain unchanged after failed duplicate add
        let config = get_config(deps.as_ref()).unwrap();
        assert_eq!(expected_tokens, config.xrpl_tokens);
    }

    #[test]
    fn test_query_xrpl_tokens() {
        let mut deps = mock_dependencies();
        let env = mock_env();

        let xrpl_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_CORE_CURRENCY.to_string(),
            issuer: TEST_XRPL_CORE_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: TEST_MULTIPLIER_1_0.to_string(),
        }];

        let mut msg = init_msg();
        msg.xrpl_tokens = xrpl_tokens.clone();

        let info = mock_info(TEST_OWNER, &[]);
        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), msg);
        assert!(res.is_ok());

        // query tokens via config
        let config_response = get_config(deps.as_ref()).unwrap();
        assert_eq!(xrpl_tokens, config_response.xrpl_tokens);
    }

    #[test]
    fn test_instantiate_with_xrpl_tokens() {
        let mut deps = mock_dependencies();
        let env = mock_env();

        let xrpl_tokens = vec![
            crate::msg::XRPLToken {
                currency: TEST_XRPL_CORE_CURRENCY.to_string(),
                issuer: TEST_XRPL_CORE_ISSUER.to_string(),
                activation_date: TEST_ACTIVATION_DATE,
                multiplier: TEST_MULTIPLIER_1_0.to_string(),
            },
            crate::msg::XRPLToken {
                currency: TEST_XRPL_XCORE_CURRENCY.to_string(),
                issuer: TEST_XRPL_XCORE_ISSUER.to_string(),
                activation_date: TEST_ACTIVATION_DATE,
                multiplier: TEST_MULTIPLIER_1_0.to_string(),
            },
            crate::msg::XRPLToken {
                currency: TEST_XRPL_SOLO_CURRENCY.to_string(),
                issuer: TEST_XRPL_SOLO_ISSUER.to_string(),
                activation_date: TEST_ACTIVATION_DATE,
                multiplier: TEST_MULTIPLIER_1_25.to_string(),
            },
        ];

        let mut msg = init_msg();
        msg.xrpl_tokens = xrpl_tokens.clone();

        let info = mock_info(TEST_OWNER, &[]);
        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), msg);
        assert!(res.is_ok());

        // verify config has correct values
        let config = get_config(deps.as_ref()).unwrap();
        assert_eq!(xrpl_tokens, config.xrpl_tokens);
    }

    #[test]
    fn test_validate_xrpl_token_invalid_currency() {
        let mut deps = mock_dependencies();
        let env = mock_env();
        let info = mock_info(TEST_OWNER, &[]);

        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), init_msg());
        assert!(res.is_ok());

        // Test invalid currency length
        let invalid_tokens = vec![crate::msg::XRPLToken {
            currency: "123".to_string(), // Too short
            issuer: TEST_XRPL_CORE_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: TEST_MULTIPLIER_1_0.to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), invalid_tokens);
        assert!(matches!(
            res.unwrap_err(),
            ContractError::InvalidXRPLCurrency { .. }
        ));

        // Test invalid currency characters
        let invalid_tokens = vec![crate::msg::XRPLToken {
            currency: "GGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG".to_string(), // Invalid hex
            issuer: TEST_XRPL_CORE_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: TEST_MULTIPLIER_1_0.to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), invalid_tokens);
        assert!(matches!(
            res.unwrap_err(),
            ContractError::InvalidXRPLCurrency { .. }
        ));
    }

    #[test]
    fn test_validate_xrpl_token_invalid_issuer() {
        let mut deps = mock_dependencies();
        let env = mock_env();
        let info = mock_info(TEST_OWNER, &[]);

        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), init_msg());
        assert!(res.is_ok());

        // Test issuer that doesn't start with 'r'
        let invalid_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_CORE_CURRENCY.to_string(),
            issuer: "aSEP47QAwU6jsZU493znUD2iGNHDQEyvA".to_string(), // Doesn't start with 'r'
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: TEST_MULTIPLIER_1_0.to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), invalid_tokens);
        assert!(matches!(
            res.unwrap_err(),
            ContractError::InvalidXRPLIssuer { .. }
        ));

        // Test issuer with invalid length
        let invalid_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_CORE_CURRENCY.to_string(),
            issuer: "rShort".to_string(), // Too short
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: TEST_MULTIPLIER_1_0.to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), invalid_tokens);
        assert!(matches!(
            res.unwrap_err(),
            ContractError::InvalidXRPLIssuer { .. }
        ));

        // Test issuer with invalid characters (contains '0')
        let invalid_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_CORE_CURRENCY.to_string(),
            issuer: "r000000000000000000000000000000".to_string(), // Contains '0'
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: TEST_MULTIPLIER_1_0.to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), invalid_tokens);
        assert!(matches!(
            res.unwrap_err(),
            ContractError::InvalidXRPLIssuer { .. }
        ));
    }

    #[test]
    fn test_validate_xrpl_token_invalid_multiplier() {
        let mut deps = mock_dependencies();
        let env = mock_env();
        let info = mock_info(TEST_OWNER, &[]);

        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), init_msg());
        assert!(res.is_ok());

        // Test empty multiplier
        let invalid_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_CORE_CURRENCY.to_string(),
            issuer: TEST_XRPL_CORE_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: "".to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), invalid_tokens);
        assert!(matches!(
            res.unwrap_err(),
            ContractError::InvalidMultiplier { .. }
        ));

        // Test negative multiplier
        let invalid_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_CORE_CURRENCY.to_string(),
            issuer: TEST_XRPL_CORE_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: "-1.0".to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), invalid_tokens);
        assert!(matches!(
            res.unwrap_err(),
            ContractError::InvalidMultiplier { .. }
        ));

        // Test zero multiplier
        let invalid_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_CORE_CURRENCY.to_string(),
            issuer: TEST_XRPL_CORE_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: "0".to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), invalid_tokens);
        assert!(matches!(
            res.unwrap_err(),
            ContractError::InvalidMultiplier { .. }
        ));

        // Test multiplier below minimum (0.1)
        let invalid_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_CORE_CURRENCY.to_string(),
            issuer: TEST_XRPL_CORE_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: "0.05".to_string(), // Below 0.1
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), invalid_tokens);
        assert!(matches!(
            res.unwrap_err(),
            ContractError::InvalidMultiplier { .. }
        ));

        // Test multiplier above maximum (10.0)
        let invalid_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_CORE_CURRENCY.to_string(),
            issuer: TEST_XRPL_CORE_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: "10.1".to_string(), // Above 10.0
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), invalid_tokens);
        assert!(matches!(
            res.unwrap_err(),
            ContractError::InvalidMultiplier { .. }
        ));

        // Test invalid characters in multiplier
        let invalid_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_CORE_CURRENCY.to_string(),
            issuer: TEST_XRPL_CORE_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: "1.2.3".to_string(), // Multiple decimal points
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), invalid_tokens);
        assert!(matches!(
            res.unwrap_err(),
            ContractError::InvalidMultiplier { .. }
        ));
    }

    #[test]
    fn test_validate_xrpl_token_valid_formats() {
        let mut deps = mock_dependencies();
        let env = mock_env();
        let info = mock_info(TEST_OWNER, &[]);

        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), init_msg());
        assert!(res.is_ok());

        // Test valid tokens with various valid formats
        let valid_tokens = vec![
            crate::msg::XRPLToken {
                currency: "434F524500000000000000000000000000000000".to_string(), // Uppercase hex
                issuer: TEST_XRPL_CORE_ISSUER.to_string(),
                activation_date: TEST_ACTIVATION_DATE,
                multiplier: "1.0".to_string(),
            },
            crate::msg::XRPLToken {
                currency: "434f524500000000000000000000000000000000".to_string(), // Lowercase hex
                issuer: TEST_XRPL_XCORE_ISSUER.to_string(),
                activation_date: TEST_ACTIVATION_DATE,
                multiplier: "1.25".to_string(),
            },
            crate::msg::XRPLToken {
                currency: "1234567890ABCDEFabcdef000000000000000000".to_string(), // Mixed case hex
                issuer: TEST_XRPL_SOLO_ISSUER.to_string(),
                activation_date: TEST_ACTIVATION_DATE,
                multiplier: "2.5".to_string(),
            },
        ];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), valid_tokens);
        assert!(res.is_ok());
    }

    #[test]
    fn test_validate_multiplier_range() {
        let mut deps = mock_dependencies();
        let env = mock_env();
        let info = mock_info(TEST_OWNER, &[]);

        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), init_msg());
        assert!(res.is_ok());

        // Test minimum boundary (0.1) - should be valid
        let valid_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_CORE_CURRENCY.to_string(),
            issuer: TEST_XRPL_CORE_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: "0.1".to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), valid_tokens);
        assert!(res.is_ok());

        // Test maximum boundary (10.0) - should be valid
        let valid_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_XCORE_CURRENCY.to_string(),
            issuer: TEST_XRPL_XCORE_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: "10.0".to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), valid_tokens);
        assert!(res.is_ok());

        // Test value just below minimum (0.099) - should be invalid
        let invalid_tokens = vec![crate::msg::XRPLToken {
            currency: TEST_XRPL_SOLO_CURRENCY.to_string(),
            issuer: TEST_XRPL_SOLO_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: "0.099".to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), invalid_tokens);
        assert!(matches!(
            res.unwrap_err(),
            ContractError::InvalidMultiplier { .. }
        ));

        // Test value just above maximum (10.01) - should be invalid
        let invalid_tokens = vec![crate::msg::XRPLToken {
            currency: "1234567890ABCDEFabcdef000000000000000000".to_string(),
            issuer: TEST_XRPL_CORE_ISSUER.to_string(),
            activation_date: TEST_ACTIVATION_DATE,
            multiplier: "10.01".to_string(),
        }];

        let info = mock_info(TEST_OWNER, &[]);
        let res = add_xrpl_tokens(deps.as_mut(), info.clone(), invalid_tokens);
        assert!(matches!(
            res.unwrap_err(),
            ContractError::InvalidMultiplier { .. }
        ));
    }
}
