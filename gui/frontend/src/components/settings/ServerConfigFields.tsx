interface ServerConfigFieldsProps {
  serverAddress: string;
  onServerAddressChange: (val: string) => void;
  deviceName: string;
  onDeviceNameChange: (val: string) => void;
  disabled: boolean;
}

export const ServerConfigFields = ({
  serverAddress,
  onServerAddressChange,
  deviceName,
  onDeviceNameChange,
  disabled,
}: ServerConfigFieldsProps) => {
  return (
    <>
      <label className="popup-label">Relay Server URL</label>
      <input
        type="text"
        className="popup-input"
        value={serverAddress}
        onChange={(e) => onServerAddressChange(e.target.value)}
        placeholder="e.g. https://relay.example.com"
        disabled={disabled}
      />

      <label className="popup-label">Device Identifier</label>
      <input
        type="text"
        className="popup-input"
        value={deviceName}
        onChange={(e) => onDeviceNameChange(e.target.value)}
        placeholder="e.g. netconnect-laptop"
        disabled={disabled}
      />
    </>
  );
};
