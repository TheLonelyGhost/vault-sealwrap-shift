# Vault Sealwrap Shifter

> [!warning]
>
> This tool is still under active development and should not be used on production systems. Use at your own peril!

## Context

In a scorched-earth disaster recovery scenario featuring HashiCorp Vault, let's assume the following:

1. The same code can be executed to stand up a new Vault cluster in a different landing zone
    - if AWS Account A was compromised, a new AWS account was created and infra is spun up there (AWS-specifics are not necessary)
2. All contact with old infrastruction prior to the cyber incident is completely impossible, with 2 exceptions:
    - Access to the underlying key materials used for the auto-unseal mechanism (e.g., stored in USB drive and shoved into locked filing cabinet)
    - Access to a recent snapshot of Vault's Integrated Storage
3. Full access to the new environment is provided to the operator in the name of quickly getting the business operational again

In this scenario, an operator might push the same key materials to a new AWS KMS key (for example) and update Vault's config file. Then, after that is configured, they'd restore from snapshot and expect it to just work, right? Same encryption key materials, same encryption method used (KMS is still just KMS, right?), so it should be the same result.

Wrong.

Unfortunately some seal mechanisms embed specific references in the Vault data which cannot be updated through normal means. In the case of AWS KMS unseal, the ARN of the original seal mechanism is embedded in several BoltDB entries. Following the above recovery strategy will still have the new Vault cluster reaching out to the old KMS key unsuccessfully. It will also trigger an attempt to do a seal migration to the new KMS key, assuming these key materials are different.

In reality we don't want Vault to attempt a seal migration, we want to forcibly assert that they are the same key materials just living in a different location.

## Objective

This tool does surgery on those BoltDB entries and updates the KMS Key ARN (if AWS KMS was used) to be set to something else. Depending on the implementation in the [go-kms-wrapping](https://github.com/hashicorp/go-kms-wrapping) library, the Key ID could be the KMS key ARN, the HSM key label, or any number of other sources.

This tool forces you to specify both the current value (as a method of confirming intent to change the value) and the desired new value, allowing you to update that in-place in the `vault.db` file contents. After doing surgery on the `vault.db` file, we clear out the copies on other Vault nodes and let Vault's inter-node replication within the cluster do its job to replicate it around.

> [!warning]
>
> This tool may only be used to migrate within the same auto-unseal technology (e.g.,
> AWS KMS key to a different AWS KMS key ARN) and must contain the same underlying
> key materials.

## Development

Requires:

- [`go-task`](https://taskfile.dev)
- [Go](https://www.golang.org/) v1.21 or higher

Testing will require a `vault` binary of some sort, whether Community Edition or Enterprise Edition (license required) is immaterial. Also access to one of the supported auto-unseal devices supported in that version of HashiCorp Vault.

To build:

```
~/workspace $ task build
task: [build] go build -trimpath -ldflags='-s -w' -v -o out/vault-sealwrap-shift .
```

## Usage

Steps:

1. Stop all Vault Server processes in the Vault cluster
2. Delete `vault.db` file from all Vault nodes except 1 (hereafter the current machine)
3. Make a backup of `vault.db` file (e.g., `cp vault.db{,.bak}`)
4. Update Vault seal configuration in Vault Server's config files to point to new unseal key
5. Run this tool against `vault.db` (found in the path to Raft Integrated Storage location configured in Vault's config files)
6. Start Vault Server process on current machine
7. Start Vault Server process on all other machines in the Vault cluster

By taking this approach, all nodes in the cluster will gossip around this same version of the Vault data to all nodes.

Once verification is complete, all nodes should have full access to all data in this restored environment. The backup (`vault.db.bak`) may then safely be deleted.

```
~/workspace $ ./out/vault-sealwrap-shift -help

Usage of ./out/vault-sealwrap-shift:
  -config string
        Path to a file containing old and new key mappings (default "./migrate.json")
  -file string
        Path to the Integrated Storage vault.db file (default "./vault.db")

~/workspace $ ./out/vault-sealwrap-shift -config ./migrate.json -file ./vault.db

Updated key info for 'core/hsm/barrier-unseal-keys'
{
  "ciphertext": "4/SLigqjr3vGW6CE1KT2hQi2PZESGB0cH8Tj37sgf38yHXIFHCWqg7bh71YqIUQxVnm+nL7bivXeN5aw2y0Msw==",
  "iv": "uyl6kR7aUjCm0/fFXJPJVg==",
  "hmac": "4Vks8mc0H5TXytzGjTf24Lsk1U2mx6AAdRFT3jbRD0o=",
  "key_info": {
    "mechanism": 4229,
    "hmac_mechanism": 593,
    "key_id": "VaultUnseal",
    "hmac_key_id": "VaultHMAC"
  }
}
Updated key info for 'core/keyring'
{
  "ciphertext": "KU6vYrDIHDY0kDf8p508e4NJl0xZxtRtnFj/M8ZTVcZ2rVUDJSlz8LJ13deo4645QotOE9+0Rm1pKyrlCKy8XQA/5G1pWNEy3fSqb44g64Rr9XGB2VIdFG5J454E4ajWuKOv95yChzqhEer32u5UB5rEGF8Eec8VJ+ZMS/2wZ5nxVGYsMvXQsOzECLrmN3BcXtE+5SM5WIRZxkJMQ+JoG3LCe6QXM5TdJs+9Hzn1NsOHY++b+DXYjQlGu0jpYKFP6byAIqxwPP46D+ovS7qxlpgSHu6AUco8+oMcS2NIV8AeZC2CfzjZxt0+a1/G8hMOmgmUwOBYHW7JB2OEBh+NgQ99pFZK/2wIxXRDmIyCP+GXMiilRZU34xZdCvqP9ijmWFGD5QGYCyTrcKVQ8wGYtGXY+bVEqlv1vczuu7EC69HhNMyVvdaewcElxI4u/vIz",
  "iv": "Zr1UN1oz4HzwLfhgARNokg==",
  "hmac": "4wAJWHlt0a6GeJV1uoD82Sb1VS8dh9KK10qbOjbVXIQ=",
  "wrapped": true,
  "key_info": {
    "mechanism": 4229,
    "hmac_mechanism": 593,
    "key_id": "VaultUnseal",
    "hmac_key_id": "VaultHMAC"
  }
}
Updated key info for 'core/leader/c8731d10-cea3-98ae-9c57-da016ebe445f'
{
  "ciphertext": "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",
  "iv": "8i394Bgkx5UfatBSyOOTng==",
  "hmac": "4CSuykgUHguXfPhwy25ZnV8w4d6fXd4brkQAiv/fsoo=",
  "wrapped": true,
  "key_info": {
    "mechanism": 4229,
    "hmac_mechanism": 593,
    "key_id": "VaultUnseal",
    "hmac_key_id": "VaultHMAC"
  }
}
Updated key info for 'core/master'
{
  "ciphertext": "eKtRFhI39LOOz9us5Ft/EwBxEHFI50BxB35xLMpdTGZrjkTP6bzqmEmJh5ToxPo9nyRWWmhj0h92blVKyAW6UI4gozu8i+nBWBljekZ/hwSQBd+NBloNgiakMoDPKeOrQ0jV75U3FxCF0tIUAS2TBgJsvYRbP4+OWUuESm5IaA9OLeZMBvNjWrvvKs9qW2Y0r4y6SybGx/W5J5YDb9jtNQ==",
  "iv": "6/IFNKRZkmga6CjASCPj9g==",
  "hmac": "3uzxEyzF6j1Mlt8900t54vmh09+PZPSqGB0sTWSHxKI=",
  "wrapped": true,
  "key_info": {
    "mechanism": 4229,
    "hmac_mechanism": 593,
    "key_id": "VaultUnseal",
    "hmac_key_id": "VaultHMAC"
  }
}
Updated key info for 'core/recovery-key'
{
  "ciphertext": "xneo8tcIWOkj6gDOAxno10Vs6i9+VouPZnnRD5Q1eO+Qm30vSOTrCWI4dY5Ii9O3",
  "iv": "vT7TWCC84cdDnn0t84H/kw==",
  "hmac": "NoxcJzeh86jLAKk+wRRexaOIi2m7rdl1RWwjbRFWRLk=",
  "wrapped": true,
  "key_info": {
    "mechanism": 4229,
    "hmac_mechanism": 593,
    "key_id": "VaultUnseal",
    "hmac_key_id": "VaultHMAC"
  }
}
Updated key info for 'core/wrapping/jwtkey'
{
  "ciphertext": "g+0AA6XZMfDBThlE5arcWLL6TLu6J/dqda+AarNEZhigxAEOtyF6X6eAys0hERwLuzQes3aYg3MpVgLMKyt+FWXuRdu3TJuTXHCrswf/5PjSK7mxp6rmzZATWiFKzPhigB1exp+lXIobdOnZsdEg83yi6V9/NbOyyMr0suHDP9Yl6GYx1gdnM1z9kGWoRhVlkNv0jKEyEbnl7nNymP8xwDfA7Ey8CTx3zFnFdIdvZ0zz+rUTpLpwbBo66/RoMimk7LlhsR+BLWIo4evsivtXXGezhEQqONzH1x47M01f5VHXnTY0WilI/w4mMvb3Ra8MBsNcoVaBtrKDjTwJsBXmyjxxJwcek832O97IRpbsEts4wLcXsdiqs1Q61y+lQE5zlLHF093kZR+z75ELCp/rlCgN0ZFSP5HzZfY+2uahadY/soLLIho/VAgvt+Zud5ZKqcXtl+omriE+RFa5fcEStMOIQUHUFvAvPCzYg4xVbY++NImEBGiPskM8OnNaV9sb2auSW9h0hfLMcaa6ZFzSXxp1WVolqIEszNiPprg9J0yt/Cguam1R7sodzZvYu7D/Fel5MXnBWTNhNWjPqtidzlBSEOuTFXd8VIUQhTYr4B7pJbE8xjcVuHMtbcOlcQEpnmHPalfutQyDADyFxLZ8YSap2L/gRvlU08CfuGV9IpjyFUE7VzaYmPKeWU0TzdBpi8xN639FvZxsrzvia4FdJQ==",
  "iv": "HtaPJiJmv6LTzLH0lIw5WQ==",
  "hmac": "AutfR064GZxMeh9N/euqZ4BOAjdv6bic+aMn7fGMvys=",
  "wrapped": true,
  "key_info": {
    "mechanism": 4229,
    "hmac_mechanism": 593,
    "key_id": "VaultUnseal",
    "hmac_key_id": "VaultHMAC"
  }
}
Updated key info for 'sys/managed-key-registry/key-config/20802be3-f8c2-4bcc-cdeb-f9ab7f2bb540'
{
  "ciphertext": "yuycSo/z4WmODAbBBDFGmFaI6fq/Rx8qVjedujXKguWEV11jne9nuHftV8l8ukAKW+YtadSI74Rh2iMhylyYmIREL2+ndQdKJ5KD/3Qbtc+vyE5xiKL01UeO1SXGJTtaDYCOEuye7Bz0r7uzkuS+EQAgtUuMnWxtPiXQOZXeQONElus5p9xkeuM/OFmzuwIdDnmBsEZcgW8cb9GIgcQW9uGs4jhd1qDFeouqWwVW93WrF7YY+0ifqwbYZLoYxoS98EexcmfO/mGhwA/mTQeXYL+V6NmIPFSKxvHAq9bQ+jbRgR7FcB2pYeX3mIiD+8OCo1yy+tZUEqLpevVhPjJDowRqqHf6gksVwviZT11Xd44lnC6fwDXb2YcDWhPueCduaVb1k7Uz7xqVqwLmHJzO0QLGyQyxLs0MRj1gOwjb0NnqFm3UJAw1hU23tMeIurNZAa0kaT1HnFSlOVc9XIb7vbbwheWWnj82B0IU5u79RTCMF6Jrilh2eudFlXidXjsTaSwQwi3L9ObIB8w/QG1pHM4RwH75Qc3Mv5SK6ln289/ZbQEM4zEZqMs2bSlQXIwHQFGUmgA5ubVo2M0220hSSD0l3cVp5D4P9/E8/S7hCW58Tiny54FkNjCtDhmKn79jbCh3waeRQzVhKsAb+inG6B/wjmSYER+E8P+2XacrmrV04XPg2fLJUX1s///ugOBrYQvF/JgXYTH5tlzMb2O72kyZGOTsr9m6QEHFWg5FauI=",
  "iv": "/fsCJJNneA9qZkUsPAuNZQ==",
  "hmac": "nwDsh5eZnVQMJQDhMI1nKIa9F8I02rB03oNab7dF4+o=",
  "wrapped": true,
  "key_info": {
    "mechanism": 4229,
    "hmac_mechanism": 593,
    "key_id": "VaultUnseal",
    "hmac_key_id": "VaultHMAC"
  }
}
Updated key info for 'sys/managed-key-registry/key-config/f076eb89-7072-1e46-b1aa-d73b6a3e7775'
{
  "ciphertext": "t8Hw4dRKcCKBoIiZ8xTdw1XLXj0SitlzDKGRfRL8Pmu8GKTmco9j2yBNUWS51oXjFTFsDHyxzADloVkAKXVeyzzKdvIuMfGqtPJ018ISKMNEwIFnJqV1n1NfJ3CsqUK8N4Gm62rLD39bw/5029gPrCICMGzpgAl9PF6kAoHK7DQNoO67FJQ7pqiHVDD6DXmPMzz3uNqG/dyEiYKr0mNIFII2L+DisBs2Ybj//5EhoZ9F93E4qRz16/sWVHWQHiLloSc+Wl0ehYMH5J8vihFgmOFQptWUQPmZz66O4VLuSrzzd2zJwOROwdwOsMGuWtaRk1+nGrf5xlS6VDDgtFA7q6tNcZW17CYnAsLgOSEML3dyGNYT4HWNBI0UHj/IDpvEKWDSYRwL00DnbpwK2m3uYqXHxXjw8LglJrXssih2BNdLLijjKpcI9xhQY0Uf+9YmNzlLhD0NRK7ZBcKripVanJSrAqrOK+brpOenmsg2zQ6GcxWpwz9f9qLyuw5UBgstPs0s38AM07n/STEpp41R0GnA18GYH6Etn/OfV6KFcTlgN3v7pLjMh9sRnpOL4WvpP5QSlJVSaU3GNov0EKNGzcnvxVb3pHcCLzR/T1S8t7E6Hq4pihJ7KfV88jZKQhqZ/L9tEl9pEo3TeG73UjtV7pvSzsy2+n9DQgu8TGDBdVsupxMNGMmQgqtxBpppqfKV7zrjeKtROUvRL1Yok0FD7j9WIOjXJCuk7MNQhIBzGKc=",
  "iv": "BeeKD/rhnVPNfDM1DSRWIA==",
  "hmac": "6isRRU1GYM3sEXOuHKDOpw9cld4EkdtqVXyJ9mHEZH8=",
  "wrapped": true,
  "key_info": {
    "mechanism": 4229,
    "hmac_mechanism": 593,
    "key_id": "VaultUnseal",
    "hmac_key_id": "VaultHMAC"
  }
}
Updated key info for 'sys/managed-key-registry/name-index/example-key'
{
  "ciphertext": "n1onfrn3wJn5Mh2YT7QXGHtf0JEu8tB1YHtn7+XD3BggKjm1Pjra4rVDlbRhVRGRjcKnss2wezw1xcD2so06L+fBD49wGFzaCe7qYWysFxZ799cgpNPb8H3M29EEkcH9o5V7dH0ZgJr/xNgAo9No4EtbinMOGLe/jW/gmcd9Lzk=",
  "iv": "J1nWwTjil2mgeBAkksQy6A==",
  "hmac": "AoQ0QRo8dDc0EI+1pzD3GZhsxB66QJ1sShC4XHxzaFI=",
  "wrapped": true,
  "key_info": {
    "mechanism": 4229,
    "hmac_mechanism": 593,
    "key_id": "VaultUnseal",
    "hmac_key_id": "VaultHMAC"
  }
}
Updated key info for 'sys/managed-key-registry/name-index/other-key'
{
  "ciphertext": "vG4l4EgmU+ewtQnLiK1BZ1fNVLRUvZWF2zdBro27FI94jJHFY5k5ajnQzlx/6TOfPdIzs2mzUQ1z50i5J8dRNY8Dko+mcE8wM2HsIHQZzpiKm2+agseLH1jNfTzCMFu8xKnVXH0jnvSD/upfEOFEsQEZsPbqnjM+DdCUA9y2+7Q=",
  "iv": "p22xVyKqY/kRgdonhmmAsw==",
  "hmac": "G69HKqprP52Ldsk++G/RjynRrPkkeTRKVVw+Pp6OUfc=",
  "wrapped": true,
  "key_info": {
    "mechanism": 4229,
    "hmac_mechanism": 593,
    "key_id": "VaultUnseal",
    "hmac_key_id": "VaultHMAC"
  }
}
Updated key info for 'sys/token/id/ha58e3d942f78464bfaaa8559bd5dceb4f5a981d7ccd4ae1d079f2540b24c7f51'
{
  "ciphertext": "Se2SZFTkrxFEk3K1hli1Z7P6QCD+WPiJPuMyN658hwGapRM0tKVnqoHbNc7/npCd8OdNJWB6rfZux8YYE0wys2vhtHw8repjO3Z5z5HqbDyopQzi5Wi+kb3lyPiTNqNzMW8eDHYaKA8r2f7J9/vYu1gOhoI4lCCvz/+hIOLjJYN8+z/rA/Lnefl5JuGO+FiK5ctOrjau4yhrQ6v7R3pLLJzAizviunpfRoGhq9oQQ19eNlmSd1qgheNmSKTIqQ/gokEjVHEkgvPBu04dGK8o729qiFMoez1OFpL0GXxKlD4dpqdWKFBTGy4MVGWyROe4lNWYAta9TX/14NVo6KHdSuWxP0qfCztrcR1ZU2FdIEgPr0flGlFxLCEq5lqucN8efZdJHRxg/b1+Vv05BBQykesIL9kl4UDWQVEHbfetPIHbnuw4cbKFLyVx1+0LA7jWuSPKz+FslVLgiYH/b6K9rtcrb2eJCZ6hduCzwlNk2xgp/DDO1ryZKLYVYPwmTW/CtI65FuE2aiqTTsxgGXHxgywJM1SsubRRALeiHDQx0BE2xmx/1hp7xl3T3ohl00RhGdUAFDrp9MlIjOGhsK9w/NrAUGHoLnQcsouB6tuC8QCSAO9KhVxV7/FQdDMnIj9MLKkDL5UOWmyy/9oXdXP1o4RWkwL8wG5m0FH6TYcGv2IKnqvyVwFQh3tiksVy69srC5Y/2MjjjoYDi/VHwv6ZuA==",
  "iv": "Z5XTGy4rkUg9PZ+iqCRifA==",
  "hmac": "MZ3uomZQTzTfsdkGvrMjeFWfbmQEsLH0mZtnqJ/bgy0=",
  "wrapped": true,
  "key_info": {
    "mechanism": 4229,
    "hmac_mechanism": 593,
    "key_id": "VaultUnseal",
    "hmac_key_id": "VaultHMAC"
  }
}
```
