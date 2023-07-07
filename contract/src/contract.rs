use cosmwasm_std::{
    entry_point, to_binary, Addr, BankMsg, Binary, Coin, CosmosMsg, Deps, DepsMut, Empty, Env,
    MessageInfo, Order, Response, StdResult, Uint128,
};
use cw2::set_contract_version;
use cw_storage_plus::Map;
use cw_utils::one_coin;

use crate::error::ContractError;
use crate::msg::{
    Config, ExecuteMsg, InstantiateMsg, PendingTransaction, PendingTransactions, QueryMsg,
    SentTransaction, SentTransactions, Transaction,
};
use crate::state::{
    MAX_AMOUNT, MIN_AMOUNT, OWNER, PENDING_TRANSACTIONS, SENT_TRANSACTIONS, THRESHOLD,
    TRUSTED_ADDRESSES,
};

const CONTRACT_NAME: &str = env!("CARGO_PKG_NAME");
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

const DEFAULT_PAGE_LIMIT: u32 = 500;
const MAX_PAGE_LIMIT: u32 = DEFAULT_PAGE_LIMIT;

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
    deps.api.addr_validate(msg.owner.as_str())?;
    OWNER.save(deps.storage, &msg.owner)?;

    if msg.threshold == 0 || msg.threshold > msg.trusted_addresses.len() as u32 {
        return Err(ContractError::InvalidThreshold {});
    }

    for addr in &msg.trusted_addresses {
        deps.api.addr_validate(addr.as_str())?;
        if TRUSTED_ADDRESSES.has(deps.storage, addr.clone()) {
            return Err(ContractError::DuplicatedTrustedAddress {});
        }
        TRUSTED_ADDRESSES.save(deps.storage, addr.clone(), &Empty {})?;
    }

    THRESHOLD.save(deps.storage, &msg.threshold)?;
    MIN_AMOUNT.save(deps.storage, &msg.min_amount)?;
    MAX_AMOUNT.save(deps.storage, &msg.max_amount)?;

    Ok(Response::new())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::ThresholdBankSend {
            id,
            amount,
            recipient,
        } => threshold_bank_send(deps, info, id, amount, recipient),
        ExecuteMsg::Withdraw {} => withdraw(deps, env, info),
        ExecuteMsg::ExecutePending { evidence_id } => execute_pending(deps, info, evidence_id),
        ExecuteMsg::UpdateMinAmount { min_amount } => update_min_amount(deps, info, min_amount),
        ExecuteMsg::UpdateMaxAmount { max_amount } => update_max_amount(deps, info, max_amount),
    }
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

    if !TRUSTED_ADDRESSES.has(deps.storage, info.sender.clone()) {
        return Err(ContractError::Unauthorized {});
    }

    if MIN_AMOUNT.load(deps.storage)?.gt(&amount.amount) {
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
    if tx.evidence_providers.len() as u32 == THRESHOLD.load(deps.storage)?
        && MAX_AMOUNT.load(deps.storage)?.ge(&amount.amount.clone())
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
    match PENDING_TRANSACTIONS.may_load(deps.storage, evidence_id.clone())? {
        None => Err(ContractError::TransactionNotFound {}),
        Some(tx) => {
            if (tx.evidence_providers.len() as u32) < THRESHOLD.load(deps.storage)? {
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
    let owner = OWNER.load(deps.storage)?;
    if info.sender != owner {
        return Err(ContractError::Unauthorized {});
    }

    MIN_AMOUNT.save(deps.storage, &min_amount)?;
    Ok(Response::new().add_attribute("min_amount", min_amount))
}

pub fn update_max_amount(
    deps: DepsMut,
    info: MessageInfo,
    max_amount: Uint128,
) -> Result<Response, ContractError> {
    let owner = OWNER.load(deps.storage)?;
    if info.sender != owner {
        return Err(ContractError::Unauthorized {});
    }

    MAX_AMOUNT.save(deps.storage, &max_amount)?;
    Ok(Response::new().add_attribute("max_amount", max_amount))
}

pub fn withdraw(deps: DepsMut, env: Env, info: MessageInfo) -> Result<Response, ContractError> {
    let contract_balances = deps.querier.query_all_balances(env.contract.address)?;
    let owner = OWNER.load(deps.storage)?;
    if info.sender != owner {
        return Err(ContractError::Unauthorized {});
    }

    let bank_send_msg: CosmosMsg = BankMsg::Send {
        to_address: owner.clone().into(),
        amount: contract_balances,
    }
        .into();

    Ok(Response::new()
        .add_attribute("recipient", owner)
        .add_message(bank_send_msg))
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

fn get_config(deps: Deps) -> StdResult<Config> {
    let owner = OWNER.load(deps.storage)?;
    let trusted_addresses: Vec<Addr> = TRUSTED_ADDRESSES
        .keys(deps.storage, None, None, Order::Ascending)
        .map(|v| v.unwrap())
        .collect();
    let threshold = THRESHOLD.load(deps.storage)?;

    let min_amount = MIN_AMOUNT.load(deps.storage)?;
    let max_amount = MAX_AMOUNT.load(deps.storage)?;

    Ok(Config {
        owner,
        trusted_addresses,
        threshold,
        min_amount,
        max_amount,
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
    use cosmwasm_std::{coin, coins, StdError, Uint128};
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

            let res = instantiate(deps.as_mut(), env.clone(), info.clone(), msg.clone().into());
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
        assert_eq!(true, res.is_ok());

        let id: String = "tx_hash1".to_string();
        let denom: String = "ucore".to_string();
        let low_amount: Coin = coin(9 as u128, denom.clone());
        let amount: Coin = coin(999 as u128, denom.clone());
        let malicious_amount: Coin = coin(777 as u128, denom.clone());

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
        assert_eq!(true, res.is_ok());

        let id: String = "tx_hash1".to_string();
        let denom: String = "ucore".to_string();
        let amount: Coin = coin(20_000 as u128, denom.clone());

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
    fn test_withdraw() {
        let mut deps = mock_dependencies();
        let env = mock_env();

        let amount = coins(1000, "core");
        let info = mock_info(TEST_OWNER, &amount);
        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), init_msg());
        assert_eq!(true, res.is_ok());
        deps.querier
            .update_balance(env.clone().contract.address, amount.clone());

        let contract_balances = deps
            .as_mut()
            .querier
            .query_all_balances(env.clone().contract.address)
            .unwrap();
        assert_eq!(amount, contract_balances);

        // execute from non-owner
        let info = mock_info(TEST_TRUSTED_ADDRESS1, &[]);
        let res = withdraw(deps.as_mut(), env.clone(), info);
        assert_eq!(ContractError::Unauthorized {}, res.unwrap_err());

        // execute form the owner
        let info = mock_info(TEST_OWNER, &[]);
        let res = withdraw(deps.as_mut(), env.clone(), info);
        let res_msgs = res.unwrap().messages;
        assert_eq!(1, res_msgs.len());
        let bank_send_msg: CosmosMsg = BankMsg::Send {
            to_address: TEST_OWNER.to_string(),
            amount,
        }
            .into();
        assert_eq!(bank_send_msg, res_msgs[0].msg);
    }

    #[test]
    fn test_update_min_max_amount() {
        let mut deps = mock_dependencies();
        let env = mock_env();

        let info = mock_info(TEST_OWNER, &[]);
        let res = instantiate(deps.as_mut(), env.clone(), info.clone(), init_msg());
        assert_eq!(true, res.is_ok());

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
        assert_eq!(true, res.is_ok());
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
        assert_eq!(true, res.is_ok());
        let config = get_config(deps.as_ref()).unwrap();
        assert_eq!(new_max_amount, config.max_amount);
    }
}
