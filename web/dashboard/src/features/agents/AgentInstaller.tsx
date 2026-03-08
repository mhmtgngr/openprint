import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { DownloadIcon, ShieldIcon, CheckIcon, CopyIcon, ClipboardDocumentIcon } from '@/components/icons';

interface Installer {
  version: string;
  os: string;
  architecture: string;
  download_url: string;
  checksum: string;
  checksum_type: string;
  size: number;
  released_at: string;
  metadata?: {
    description?: string;
    minimum_os_version?: string;
  };
}

interface EnrollmentToken {
  token: string;
  organization_id: string;
  expires_at: string;
  max_uses: number;
  used_count: number;
  description: string;
  valid: boolean;
}

export const AgentInstaller = () => {
  const queryClient = useQueryClient();
  const [selectedOS, setSelectedOS] = useState<string>('windows');
  const [selectedArch, setSelectedArch] = useState<string>('amd64');
  const [copiedToken, setCopiedToken] = useState(false);
  const [copiedCommand, setCopiedCommand] = useState(false);

  // Fetch available installers
  const { data: installersData } = useQuery({
    queryKey: ['agent-installers'],
    queryFn: async () => {
      const res = await fetch('/api/v1/agent-installers');
      if (!res.ok) throw new Error('Failed to fetch installers');
      return res.json();
    },
  });

  // Fetch enrollment token
  const { data: tokenData } = useQuery({
    queryKey: ['enrollment-token'],
    queryFn: async () => {
      const res = await fetch('/api/v1/enrollment-tokens');
      if (!res.ok) throw new Error('Failed to fetch enrollment token');
      const data = await res.json();
      // Return the first valid token
      return data.tokens?.find((t: EnrollmentToken) => t.valid) || null;
    },
  });

  // Generate enrollment token mutation
  const generateTokenMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch('/api/v1/enrollment-tokens/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          max_uses: 10,
          expires_in_days: 30,
          description: 'Agent installer enrollment token',
        }),
      });
      if (!res.ok) throw new Error('Failed to generate token');
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['enrollment-token'] });
    },
  });

  const installers = installersData?.installers || [];
  const token = tokenData;

  const selectedInstaller = installers.find(
    (i: Installer) => i.os === selectedOS && i.architecture === selectedArch
  );

  const osOptions = [
    { value: 'windows', label: 'Windows', icon: '🪟' },
    { value: 'linux', label: 'Linux', icon: '🐧' },
    { value: 'macos', label: 'macOS', icon: '🍎' },
  ];

  const archOptions = [
    { value: 'amd64', label: 'x64 (64-bit)' },
    { value: 'arm64', label: 'ARM64' },
  ];

  const installCommand = token
    ? selectedOS === 'windows'
      ? `openprint-agent-installer.exe /token ${token}`
      : selectedOS === 'linux'
      ? `sudo ./install.sh --token=${token}`
      : `sudo installer -pkg OpenPrint-Agent.pkg -target /`
    : '';

  const copyToken = () => {
    if (token) {
      navigator.clipboard.writeText(token.token);
      setCopiedToken(true);
      setTimeout(() => setCopiedToken(false), 2000);
    }
  };

  const copyCommand = () => {
    if (installCommand) {
      navigator.clipboard.writeText(installCommand);
      setCopiedCommand(true);
      setTimeout(() => setCopiedCommand(false), 2000);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Install OpenPrint Agent
          </h2>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            Download and install the agent to connect your printers
          </p>
        </div>
      </div>

      {/* Enrollment Token Card */}
      <div className="bg-gradient-to-r from-blue-50 to-cyan-50 dark:from-blue-900/20 dark:to-cyan-900/20 rounded-lg p-6 border border-blue-200 dark:border-blue-800">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-blue-100 dark:bg-blue-900/40 rounded-lg text-blue-600 dark:text-blue-400">
            <ShieldIcon className="w-6 h-6" />
          </div>
          <div className="flex-1">
            <h3 className="font-semibold text-gray-900 dark:text-gray-100">Enrollment Token</h3>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
              Use this token to securely register your agent with your organization
            </p>

            {token ? (
              <div className="mt-4 space-y-3">
                <div className="flex items-center gap-2">
                  <code className="flex-1 px-3 py-2 bg-white dark:bg-gray-800 rounded border border-gray-300 dark:border-gray-600 text-sm font-mono text-gray-900 dark:text-gray-100">
                    {token.token}
                  </code>
                  <button
                    onClick={copyToken}
                    className="px-3 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-1"
                  >
                    {copiedToken ? <CheckIcon className="w-4 h-4" /> : <CopyIcon className="w-4 h-4" />}
                    {copiedToken ? 'Copied!' : 'Copy'}
                  </button>
                </div>
                <div className="text-xs text-gray-500 dark:text-gray-400 space-y-1">
                  <p>• Expires: {new Date(token.expires_at).toLocaleDateString()}</p>
                  <p>• Uses: {token.used_count} / {token.max_uses}</p>
                </div>
              </div>
            ) : (
              <button
                onClick={() => generateTokenMutation.mutate()}
                disabled={generateTokenMutation.isPending}
                className="mt-4 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white rounded-lg text-sm font-medium transition-colors"
              >
                {generateTokenMutation.isPending ? 'Generating...' : 'Generate Enrollment Token'}
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Platform Selection */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <h3 className="font-semibold text-gray-900 dark:text-gray-100">Select Platform</h3>
        </div>
        <div className="p-6 space-y-6">
          {/* OS Selection */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
              Operating System
            </label>
            <div className="grid grid-cols-3 gap-3">
              {osOptions.map((os) => (
                <button
                  key={os.value}
                  onClick={() => setSelectedOS(os.value)}
                  className={`p-4 rounded-lg border-2 transition-all ${
                    selectedOS === os.value
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                      : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
                  }`}
                >
                  <div className="text-2xl mb-2">{os.icon}</div>
                  <div className="text-sm font-medium text-gray-900 dark:text-gray-100">{os.label}</div>
                </button>
              ))}
            </div>
          </div>

          {/* Architecture Selection */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
              Architecture
            </label>
            <div className="flex gap-3">
              {archOptions.map((arch) => (
                <button
                  key={arch.value}
                  onClick={() => setSelectedArch(arch.value)}
                  className={`px-4 py-2 rounded-lg border-2 text-sm font-medium transition-all ${
                    selectedArch === arch.value
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
                      : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600 text-gray-700 dark:text-gray-300'
                  }`}
                >
                  {arch.label}
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Selected Installer Info */}
      {selectedInstaller && (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <div>
              <h3 className="font-semibold text-gray-900 dark:text-gray-100">
                {selectedInstaller.os.charAt(0).toUpperCase() + selectedInstaller.os.slice(1)} {selectedInstaller.architecture} Installer
              </h3>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                Version {selectedInstaller.version} • {selectedInstaller.metadata?.description}
              </p>
            </div>
            <a
              href={selectedInstaller.download_url}
              download
              className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
            >
              <DownloadIcon className="w-4 h-4" />
              Download
            </a>
          </div>
          <div className="p-6">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-gray-500 dark:text-gray-400">File Size:</span>
                <span className="ml-2 text-gray-900 dark:text-gray-100">
                  {selectedInstaller.size > 0 ? `${(selectedInstaller.size / 1024 / 1024).toFixed(1)} MB` : 'N/A'}
                </span>
              </div>
              <div>
                <span className="text-gray-500 dark:text-gray-400">Checksum:</span>
                <span className="ml-2 font-mono text-gray-900 dark:text-gray-100">
                  {selectedInstaller.checksum || 'N/A'}
                </span>
              </div>
              <div>
                <span className="text-gray-500 dark:text-gray-400">Released:</span>
                <span className="ml-2 text-gray-900 dark:text-gray-100">
                  {new Date(selectedInstaller.released_at).toLocaleDateString()}
                </span>
              </div>
              <div>
                <span className="text-gray-500 dark:text-gray-400">Min OS Version:</span>
                <span className="ml-2 text-gray-900 dark:text-gray-100">
                  {selectedInstaller.metadata?.minimum_os_version || 'N/A'}
                </span>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Installation Instructions */}
      {token && selectedInstaller && (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700">
            <h3 className="font-semibold text-gray-900 dark:text-gray-100">Installation Instructions</h3>
          </div>
          <div className="p-6">
            <ol className="space-y-3 text-sm text-gray-700 dark:text-gray-300">
              <li className="flex gap-3">
                <span className="flex-shrink-0 w-6 h-6 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded-full flex items-center justify-center text-xs font-semibold">1</span>
                <span>Download the installer using the button above</span>
              </li>
              <li className="flex gap-3">
                <span className="flex-shrink-0 w-6 h-6 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded-full flex items-center justify-center text-xs font-semibold">2</span>
                <span>Run the installer with administrator/root privileges</span>
              </li>
              <li className="flex gap-3">
                <span className="flex-shrink-0 w-6 h-6 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded-full flex items-center justify-center text-xs font-semibold">3</span>
                <span>When prompted, enter your enrollment token</span>
              </li>
              <li className="flex gap-3">
                <span className="flex-shrink-0 w-6 h-6 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded-full flex items-center justify-center text-xs font-semibold">4</span>
                <span>Complete the installation. The agent will automatically discover printers.</span>
              </li>
            </ol>

            {/* Command Line Install */}
            <div className="mt-6 p-4 bg-gray-100 dark:bg-gray-900 rounded-lg">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase">
                  Command Line Installation
                </span>
                <button
                  onClick={copyCommand}
                  className="text-xs text-blue-600 dark:text-blue-400 hover:underline flex items-center gap-1"
                >
                  {copiedCommand ? <CheckIcon className="w-3 h-3" /> : <ClipboardDocumentIcon className="w-3 h-3" />}
                  {copiedCommand ? 'Copied!' : 'Copy'}
                </button>
              </div>
              <code className="text-sm text-gray-900 dark:text-gray-100 block overflow-x-auto">
                {installCommand}
              </code>
            </div>
          </div>
        </div>
      )}

      {/* Requirements */}
      <div className="bg-yellow-50 dark:bg-yellow-900/20 rounded-lg p-4 border border-yellow-200 dark:border-yellow-800">
        <h4 className="font-medium text-yellow-900 dark:text-yellow-100 mb-2">Requirements</h4>
        <ul className="text-sm text-yellow-800 dark:text-yellow-200 space-y-1">
          <li>• Administrator/root privileges for installation</li>
          <li>• Network connectivity to the OpenPrint server</li>
          <li>• Valid enrollment token from your organization</li>
        </ul>
      </div>
    </div>
  );
};
