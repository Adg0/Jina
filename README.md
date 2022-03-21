# go-jina

Algorand hackathon.

## Demo
Live demo: https://github.io/
Demo video: https://youtube.com/

## About Jina

Jina NFT-Fi allows any asset(NFT) in Algorand to be leveraged.
With a 3% return for liquidity providers that give liquidity to the protocol, Jina becomes the go-to site for delivering liquidity for NFTs.
Liquidity providers will keep their liquidity asset in their account until a borrower uses it, at which point the protocol will reward them with an I-O-U stable-coin token and 3% of the borrowed amount.
Additionally, assets(NFT) that require KYC or special permissions might benefit from the pure non-custodial protocol.
Jina's leveraged NFT remains locked at the borrower's address until the loan amount is repaid, which is a pure non-custodial transaction.

### Smartcontract Interactions

There are two smartcontracts that power Jina dapp.
* Jina contract
* Liquidator contract
1. Jina Contract
Jina contract holds the state machine and locks/unlocks NFT in account (freezes/unfreezes  NFT) .
State machine, tracks:
* What NFT is used as collateral
* How much collateral is used for loan
* How much loan is borrowed
* And how much is available loan
2. Liquidator Contract
Liquidator contract reads current price of NFT from oracle and if loan is more than 90% of collateral it liquidates the NFT locked.

### Transfering Admin addresses to Jina

First step is to transfer your NFT's admin address to Jina.
This will make your NFT leverageable.
* Set manager and freeze admin address to Jina smartcontract
* Set clawback to liquidator smartcontract

### Optin to Jina

Make an application call to optin to the dapp.

### Leveraging NFT

Use your NFT as collateral, to borrow USDCa stablecoin.
You'll get requested loan amount in USDCa and your NFT will be locked.
* Set which NFT you want to collateralize
* Set amount of collateral
* Request loan

### Providing Liquidity

Choose which NFTs can borrow from your account.
* Set maximum amount you are willing to lend.
* Set expiration date for aggrement.

### Repaying loan

Send USDCa to Jina contract.
Your loan amount state will be decremented by sent repaid amount.

### Repaying full loan

Send USDCa to Jina contract.
Your collateral assets will be unfrozen.

### Claming USDCa

Send JUSD(I-O-U token of Jina contract) to Jina contract.
You'll receive a 1:1 USDCa

### Liquidation

Chose among used as collateral to liquidate
Specify the addresss to liquidate
Pay 95% of collateral's value to Jina contract
Set an account that will receive the liquidated asset
You'll be sent the collateral to the address you specified

## Building Jina Locally

## Techincal info
* NFTs used as collateral are frozen in account, only when account takes out loan.
* Frozen NFTs are unfrozen when full loan is paid back.
* There is no interest rate for borrowing USDCa.
* A 3% fee is paid to take out loan.
* Lenders sign a delegated logic signature to allow any account to withdraw USDCa that fullfill the following:
	1. Calls Jina contract
	2. Withdraws maximum of staked amount
* Any account that holds JUSD can claim 1:1 USDCa by sending the JUSD to Jina contract.
* Borrower can borrow from upto 4 lenders
* Liquidation call:
