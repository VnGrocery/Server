// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract IntegrityRegistry {
    struct Record {
        bytes32 dataHash;
        uint256 timestamp;
        uint256 version;
        bool revoked;
        bool exists;
    }

    mapping(string => Record) private latestByRecordId;
    address public owner;

    event HashCommitted(string indexed recordId, bytes32 indexed dataHash, uint256 timestamp, uint256 version);

    modifier onlyOwner() {
        require(msg.sender == owner, "not authorized");
        _;
    }

    constructor(address initialOwner) {
        owner = initialOwner;
    }

    function commitHash(string calldata recordId, bytes32 dataHash, uint256 timestamp, uint256 version) external onlyOwner {
        Record storage current = latestByRecordId[recordId];
        require(version > current.version, "version must increase");

        latestByRecordId[recordId] = Record({
            dataHash: dataHash,
            timestamp: timestamp,
            version: version,
            revoked: false,
            exists: true
        });

        emit HashCommitted(recordId, dataHash, timestamp, version);
    }

    function getLatest(string calldata recordId) external view returns (bytes32, uint256, uint256, bool) {
        Record memory current = latestByRecordId[recordId];
        if (!current.exists) {
            return (bytes32(0), 0, 0, false);
        }
        return (current.dataHash, current.timestamp, current.version, current.revoked);
    }

    function verify(string calldata recordId, bytes32 dataHash) external view returns (bool) {
        Record memory current = latestByRecordId[recordId];
        return current.exists && !current.revoked && current.dataHash == dataHash;
    }
}
