// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "forge-std/Test.sol";

interface ISimulateTxERC20 {
  function approve(address spender, uint256 amount) external returns (bool);
}

interface ISimulateTxERC721 {
  function approve(address spender, uint256 tokenId) external;
}

contract SimulateTxScript is Test {
  struct LabelOverride {
    address account;
    string label;
  }

  struct ERC20BalanceOverride {
    address token;
    address account;
    uint256 balance;
  }

  struct ERC20ApprovalOverride {
    address token;
    address owner;
    address spender;
    uint256 amount;
  }

  struct ERC721ApprovalOverride {
    address token;
    address owner;
    address spender;
    uint256 tokenId;
  }

  /**
   * @notice Override the states of specific addresses and perform the tx
   * @param labelOverrides The labels to assign to addresses in the trace
   * @param erc20BalanceOverrides The ERC20 balance overrides
   * @param erc20ApprovalOverrides The ERC20 approval overrides
   * @param erc721ApprovalOverrides The ERC721 approval overrides
   * @param stateOverridingContractBytecode The bytecode of the contract which overrides additional states
   * @param sender The sender address
   * @param target The target contract address
   * @param data The transaction data
   */
  function run(
    LabelOverride[] calldata labelOverrides,
    ERC20BalanceOverride[] calldata erc20BalanceOverrides,
    ERC20ApprovalOverride[] calldata erc20ApprovalOverrides,
    ERC721ApprovalOverride[] calldata erc721ApprovalOverrides,
    bytes memory stateOverridingContractBytecode,
    address sender,
    address target,
    bytes memory data
  ) public {
    // label addresses
    for (uint256 i = 0; i < labelOverrides.length; i++) {
      vm.label(labelOverrides[i].account, labelOverrides[i].label);
    }

    // override erc20 balances
    for (uint256 i = 0; i < erc20BalanceOverrides.length; i++) {
      deal(erc20BalanceOverrides[i].token, erc20BalanceOverrides[i].account, erc20BalanceOverrides[i].balance);
    }

    // override erc20 approvals
    for (uint256 i = 0; i < erc20ApprovalOverrides.length; i++) {
      vm.prank(erc20ApprovalOverrides[i].owner);
      ISimulateTxERC20(erc20ApprovalOverrides[i].token)
        .approve(erc20ApprovalOverrides[i].spender, erc20ApprovalOverrides[i].amount);
    }

    // override erc721 approvals
    for (uint256 i = 0; i < erc721ApprovalOverrides.length; i++) {
      vm.prank(erc721ApprovalOverrides[i].owner);
      ISimulateTxERC721(erc721ApprovalOverrides[i].token)
        .approve(erc721ApprovalOverrides[i].spender, erc721ApprovalOverrides[i].tokenId);
    }

    // override additional states
    if (stateOverridingContractBytecode.length > 0) {
      address stateOverridingContract;
      assembly {
        stateOverridingContract := create(
          0,
          add(stateOverridingContractBytecode, 0x20),
          mload(stateOverridingContractBytecode)
        )
      }

      if (stateOverridingContract == address(0)) {
        revert("Failed to deploy state overriding contract");
      } else {
        _callAndBubble(stateOverridingContract, "");
      }
    }

    // execute transaction
    vm.prank(sender);
    _callAndBubble(target, data);
  }

  function _callAndBubble(address target, bytes memory data) internal {
    (bool success, bytes memory returndata) = target.call(data);
    if (!success) {
      if (returndata.length > 0) {
        assembly {
          revert(add(returndata, 0x20), mload(returndata))
        }
      }
      revert("SimulateTxScript: call failed");
    }
  }
}
