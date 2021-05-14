// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

interface IAddressRegistry {
    function getOrakuruCoreAddr() external view returns (address);

    function getStakingAddr() external view returns (address);

    function getOrkTokenAddr() external view returns (address);
}

interface IStaking {
    function isRegisteredOracle(address _oracle) external view returns (bool);

    function registeredOraclesNum() external view returns (uint256);

    function getThresholdNum() external view returns (uint256);
}

interface IOrakuruCore {
    enum Type {MostFrequent, Median, Average}

    event Requested(
        bytes32 indexed requestId,
        string dataSource,
        string selector,
        address indexed callbackAddr,
        Type aggrType,
        uint8 precision,
        uint256 executionTimestamp
    );

    event Submitted(
        bytes32 requestId,
        string submittedResult,
        bytes parsedResult,
        address oracle
    );
    event Fulfilled(bytes32 indexed requestId, bytes result);
    event Canceled(bytes32 indexed requestId);

    struct Request {
        bytes32 id;
        string dataSource;
        string selector;
        address callbackAddr;
        uint256 executionTimestamp;
        bool isFulfilled;
        Type aggrType;
        uint8 precision;
    }

    struct Response {
        bytes32 id;
        bytes32 requestId;
        bytes result;
        address submittedBy;
        uint256 submittedAt;
    }

    function makeRequest(
        string calldata _dataSource,
        string calldata _selector,
        address _calldataAddr,
        Type _aggrType,
        uint8 _precision,
        uint256 _executionTimestamp
    ) external returns (bytes32);

    function submitResult(bytes32 _requestId, string memory _result) external;

    function fulfillRequest(bytes32 _requestId) external;

    function getPendingRequests() external view returns (bytes32[] memory);

    function cancelRequest(bytes32 _requestId) external returns (bool);

    function getResponses(bytes32 _requestId)
    external
    view
    returns (Response[] memory);

    function getResultsBytes(bytes32 _requestId)
    external
    view
    returns (bytes[] memory);

    function getResultsUint(bytes32 _requestId)
    external
    view
    returns (uint256[] memory);

    function getNonceFor(address _addr) external view returns (uint256);

    function getRequest(bytes32 _requestId)
    external
    view
    returns (
        bytes32 id,
        string memory dataSource,
        string memory selector,
        address callbackAddr,
        uint256 executionTimestamp,
        bool isFulfilled,
        Type aggrType,
        uint8 precision
    );

    function addressRegistry() external view returns (IAddressRegistry);
}
