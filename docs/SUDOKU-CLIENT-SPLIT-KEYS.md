# Per-client Sudoku split keys

A Sudoku inbound can deliver a different client configuration to each panel client without changing the server inbound or the `sing-box-extended` core.

## Configure a client

1. Keep the Sudoku inbound configured with its Master Public Key.
2. On a trusted administrator machine, use the official Sudoku CLI to derive a Split Private Key:

   ```bash
   sudoku -keygen -more <MASTER_PRIVATE_KEY>
   ```

3. Open the client editor, select **Config**, and paste the 128-character `Split Private Key` into **Sudoku client split private key**.

The panel accepts only a 64-byte split key encoded as 128 hexadecimal characters. It normalizes uppercase hex to lowercase. Do not paste a 64-character Master Private Key or the Master Public Key. The panel never asks for or stores the Master Private Key.

The personal key is used only in that client's:

- `sudoku://` link;
- QR code derived from the link;
- Sudoku outbound in JSON subscriptions.

If the field is empty or absent, delivery falls back to the inbound key, preserving existing client behavior. Backups already include `Client.Config`, so they preserve the optional split key; regenerated links use the restored value.

## Security limitations

A Split Private Key is not an independently revocable password. Each split key contains two scalar shares that reconstruct the same master scalar. A holder can recover equivalent authority and derive more split keys.

Consequences:

- deleting or disabling a panel client does not invalidate a key already imported into an application;
- removing the personal key only changes future panel delivery;
- complete revocation requires rotating the inbound master pair and reissuing keys to every remaining client;
- Sudoku traffic and online/offline state cannot be reliably attributed per panel client without protocol-adapter support.

Treat every split key as highly sensitive. Do not put it in logs, issue reports, screenshots, or chat messages.
