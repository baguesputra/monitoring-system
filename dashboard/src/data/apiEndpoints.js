// API Endpoints Data for Seneschal Collector API Documentation
// Source of truth for the API Reference page

export const apiEndpoints = [
  {
    id: 'post-metrics',
    tag: 'Agent',
    method: 'POST',
    path: '/api/metrics',
    summary: 'Receive telemetry from agent',
    description: 'Agents send CPU, RAM, Disk, Network throughput, and service status every 5-15 seconds. This is the primary ingestion endpoint for real-time telemetry.',
    auth: {
      type: 'agent-token',
      description: 'Requires valid agent token in Authorization header (planned Phase 6). Currently accepts unauthenticated requests in development.',
      headers: ['Authorization: Bearer <agent-token>', 'Content-Type: application/json']
    },
    requestBody: {
      required: true,
      contentType: 'application/json',
      schema: {
        type: 'object',
        required: ['server_id', 'cpu_percent', 'ram_percent', 'disk_percent', 'network_sent_bps', 'network_recv_bps', 'service_status', 'timestamp'],
        properties: {
          server_id: { type: 'string', description: 'Unique device identifier (e.g., "PC-HRD-01")' },
          cpu_percent: { type: 'number', format: 'float', minimum: 0, maximum: 100, description: 'CPU utilization percentage' },
          ram_percent: { type: 'number', format: 'float', minimum: 0, maximum: 100, description: 'RAM utilization percentage' },
          disk_percent: { type: 'number', format: 'float', minimum: 0, maximum: 100, description: 'Disk utilization percentage' },
          network_sent_bps: { type: 'number', format: 'float', minimum: 0, description: 'Network bytes sent per second' },
          network_recv_bps: { type: 'number', format: 'float', minimum: 0, description: 'Network bytes received per second' },
          service_status: { type: 'object', additionalProperties: { type: 'boolean' }, description: 'Map of service_name -> is_running' },
          timestamp: { type: 'string', format: 'date-time', description: 'ISO 8601 timestamp of measurement' }
        }
      },
      example: {
        server_id: "PC-HRD-01",
        cpu_percent: 45.2,
        ram_percent: 67.8,
        disk_percent: 42.1,
        network_sent_bps: 102400,
        network_recv_bps: 204800,
        service_status: { "nginx": true, "postgres": true },
        timestamp: "2026-10-01T10:30:00Z"
      }
    },
    responses: {
      200: { description: 'Telemetry received successfully', example: { status: "received" } },
      400: { description: 'Invalid JSON payload or missing required fields', example: { error: "Invalid JSON payload" } },
      401: { description: 'Invalid or missing agent token (when auth enabled)', example: { error: "Unauthorized" } },
      500: { description: 'Internal server error', example: { error: "Internal server error" } }
    },
    curl: `curl -X POST http://localhost:8081/api/metrics \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <agent-token>" \\
  -d '{
    "server_id": "PC-HRD-01",
    "cpu_percent": 45.2,
    "ram_percent": 67.8,
    "disk_percent": 42.1,
    "network_sent_bps": 102400,
    "network_recv_bps": 204800,
    "service_status": { "nginx": true, "postgres": true },
    "timestamp": "2026-10-01T10:30:00Z"
  }'`,
    tryIt: true
  },
  {
    id: 'post-asset-info',
    tag: 'Agent',
    method: 'POST',
    path: '/api/asset-info',
    summary: 'Receive hardware inventory from agent',
    description: 'Agents send complete hardware inventory (CPU, RAM, Disk, Battery, Serial, OS, Network, Applications, User accounts) every 24 hours or on startup. Performs upsert on asset_info and replaces installed_applications and device_users.',
    auth: {
      type: 'agent-token',
      description: 'Requires valid agent token in Authorization header (planned Phase 6). Currently accepts unauthenticated requests in development.',
      headers: ['Authorization: Bearer <agent-token>', 'Content-Type: application/json']
    },
    requestBody: {
      required: true,
      contentType: 'application/json',
      schema: {
        type: 'object',
        required: ['server_id'],
        properties: {
          server_id: { type: 'string' },
          processor_model: { type: 'string' },
          processor_cores: { type: 'integer', minimum: 1 },
          processor_clock_ghz: { type: 'number', format: 'float' },
          disk_type: { type: 'string', enum: ['SSD', 'HDD'] },
          disk_size_gb: { type: 'number', format: 'float' },
          disk_health_status: { type: 'string', enum: ['OK', 'WARNING', 'FAIL', 'UNKNOWN'] },
          disk_health_percent: { type: 'integer', minimum: 0, maximum: 100, nullable: true },
          ram_total_gb: { type: 'number', format: 'float' },
          ram_slots_used: { type: 'integer' },
          ram_slots_total: { type: 'integer' },
          ram_ddr_type: { type: 'string', enum: ['DDR3', 'DDR4', 'DDR5', 'Unknown'] },
          battery_health_percent: { type: 'number', format: 'float', minimum: 0, maximum: 100, nullable: true },
          serial_number: { type: 'string' },
          ip_address: { type: 'string', format: 'ipv4' },
          mac_address: { type: 'string' },
          os_version: { type: 'string' },
          applications: {
            type: 'array',
            items: {
              type: 'object',
              properties: {
                app_name: { type: 'string' },
                app_version: { type: 'string' },
                publisher: { type: 'string' },
                install_date: { type: 'string' }
              }
            }
          },
          device_users: {
            type: 'array',
            items: {
              type: 'object',
              properties: {
                username: { type: 'string' },
                full_name: { type: 'string' },
                privilege: { type: 'string', enum: ['administrator', 'standard_user', 'limited'] },
                is_disabled: { type: 'boolean' },
                is_logged_in: { type: 'boolean' }
              }
            }
          }
        }
      },
      example: {
        server_id: "PC-HRD-01",
        processor_model: "Intel Core i7-10700",
        processor_cores: 8,
        processor_clock_ghz: 2.9,
        disk_type: "SSD",
        disk_size_gb: 512,
        disk_health_status: "OK",
        disk_health_percent: 98,
        ram_total_gb: 32,
        ram_slots_used: 2,
        ram_slots_total: 4,
        ram_ddr_type: "DDR4",
        battery_health_percent: 95.5,
        serial_number: "SGH731S84K",
        ip_address: "192.168.1.100",
        mac_address: "00:1A:2B:3C:4D:5E",
        os_version: "Windows 10 Pro 10.0.19045",
        applications: [
          { "app_name": "Google Chrome", "app_version": "118.0.5993.88", "publisher": "Google LLC", "install_date": "2023-10-15" }
        ],
        device_users: [
          { "username": "IT 10", "full_name": "", "privilege": "administrator", "is_disabled": false, "is_logged_in": true }
        ]
      }
    },
    responses: {
      200: { description: 'Asset info received and upserted', example: { status: "received" } },
      400: { description: 'Invalid JSON payload or missing server_id', example: { error: "server_id tidak boleh kosong" } },
      401: { description: 'Invalid or missing agent token', example: { error: "Unauthorized" } },
      500: { description: 'Database error', example: { error: "Internal server error" } }
    },
    curl: `curl -X POST http://localhost:8081/api/asset-info \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <agent-token>" \\
  -d '{
    "server_id": "PC-HRD-01",
    "processor_model": "Intel Core i7-10700",
    "processor_cores": 8,
    "processor_clock_ghz": 2.9,
    "disk_type": "SSD",
    "disk_size_gb": 512,
    "disk_health_status": "OK",
    "disk_health_percent": 98,
    "ram_total_gb": 32,
    "ram_slots_used": 2,
    "ram_slots_total": 4,
    "ram_ddr_type": "DDR4",
    "battery_health_percent": 95.5,
    "serial_number": "SGH731S84K",
    "ip_address": "192.168.1.100",
    "mac_address": "00:1A:2B:3C:4D:5E",
    "os_version": "Windows 10 Pro 10.0.19045",
    "applications": [{ "app_name": "Google Chrome", "app_version": "118.0.5993.88", "publisher": "Google LLC", "install_date": "2023-10-15" }],
    "device_users": [{ "username": "IT 10", "full_name": "", "privilege": "administrator", "is_disabled": false, "is_logged_in": true }]
  }'`,
    tryIt: true
  },
  {
    id: 'get-servers',
    tag: 'Dashboard',
    method: 'GET',
    path: '/api/servers',
    summary: 'List all registered devices',
    description: 'Returns all registered devices with their metadata (owner, department, location, device type). Used by dashboard to populate the server list.',
    auth: {
      type: 'session',
      description: 'Requires valid dashboard session (cookie-based auth planned). Currently open in development.',
      headers: ['Cookie: session=<session-id>']
    },
    parameters: [],
    responses: {
      200: {
        description: 'Array of server objects',
        example: [
          {
            "id": 1,
            "server_id": "PC-HRD-01",
            "hostname": "PC-HRD-01",
            "ip_address": "192.168.1.100",
            "location": "Ruang IT",
            "device_type": "pc",
            "owner_name": "Bagues IT",
            "department": "IT",
            "notes": "Primary workstation",
            "is_active": true,
            "created_at": "2026-10-01T04:47:00.269786Z"
          }
        ]
      },
      401: { description: 'Unauthorized - session required', example: { error: "Unauthorized" } },
      500: { description: 'Database error' }
    },
    curl: `curl -X GET http://localhost:8081/api/servers \\
  -H "Cookie: session=<session-id>"`,
    tryIt: true
  },
  {
    id: 'get-server-metrics',
    tag: 'Dashboard',
    method: 'GET',
    path: '/api/servers/{id}/metrics',
    summary: 'Get historical metrics for a device',
    description: 'Returns time-series metrics (CPU, RAM, Disk, Network) for a specific device within a time range. Results ordered chronologically (oldest first).',
    auth: {
      type: 'session',
      description: 'Requires valid dashboard session',
      headers: ['Cookie: session=<session-id>']
    },
    parameters: [
      { name: 'id', in: 'path', required: true, type: 'string', example: 'PC-HRD-01', description: 'Server ID (e.g., "PC-HRD-01")' },
      { name: 'range', in: 'query', required: false, type: 'string', default: '1h', description: 'Time range: 30m, 1h, 6h, 24h, 7d', enum: ['30m', '1h', '6h', '24h', '7d'] }
    ],
    responses: {
      200: {
        description: 'Array of metric records (chronological order)',
        example: [
          {
            "server_id": "PC-HRD-01",
            "cpu_percent": 45.2,
            "ram_percent": 67.8,
            "disk_percent": 42.1,
            "network_sent_bps": 102400,
            "network_recv_bps": 204800,
            "recorded_at": "2026-10-01T10:25:00Z"
          }
        ]
      },
      400: { description: 'Invalid range parameter (use 30m, 1h, 6h, 24h, 7d)', example: { error: "Parameter range tidak valid" } },
      404: { description: 'Server not found', example: { error: "Server tidak ditemukan" } },
      500: { description: 'Database error' }
    },
    curl: `curl -X GET "http://localhost:8081/api/servers/PC-HRD-01/metrics?range=1h" \\
  -H "Cookie: session=<session-id>"`,
    tryIt: true
  },
  {
    id: 'get-server-status',
    tag: 'Dashboard',
    method: 'GET',
    path: '/api/servers/{id}/status',
    summary: 'Get current status for a device',
    description: 'Returns the latest metrics snapshot plus current service status for a specific device. Used for real-time status cards.',
    auth: {
      type: 'session',
      description: 'Requires valid dashboard session',
      headers: ['Cookie: session=<session-id>']
    },
    parameters: [
      { name: 'id', in: 'path', required: true, type: 'string', example: 'PC-HRD-01', description: 'Server ID (e.g., "PC-HRD-01")' }
    ],
    responses: {
      200: {
        description: 'Current device status with latest metrics and service status',
        example: {
          "server_id": "PC-HRD-01",
          "hostname": "PC-HRD-01",
          "cpu_percent": 45.2,
          "ram_percent": 67.8,
          "disk_percent": 42.1,
          "last_seen_at": "2026-10-01T10:30:00Z",
          "service_status": { "nginx": true, "postgres": false }
        }
      },
      404: { description: 'Server not found or no data yet', example: { error: "Server tidak ditemukan atau belum ada data" } },
      500: { description: 'Database error' }
    },
    curl: `curl -X GET http://localhost:8081/api/servers/PC-HRD-01/status \\
  -H "Cookie: session=<session-id>"`,
    tryIt: true
  },
  {
    id: 'get-server-asset-info',
    tag: 'Dashboard',
    method: 'GET',
    path: '/api/servers/{id}/asset-info',
    summary: 'Get hardware inventory for a device',
    description: 'Returns the latest hardware inventory (CPU, RAM, Disk, Battery, Serial, OS, Network) for a specific device.',
    auth: {
      type: 'session',
      description: 'Requires valid dashboard session',
      headers: ['Cookie: session=<session-id>']
    },
    parameters: [
      { name: 'id', in: 'path', required: true, type: 'string', example: 'PC-HRD-01', description: 'Server ID (e.g., "PC-HRD-01")' }
    ],
    responses: {
      200: {
        description: 'Complete hardware inventory',
        example: {
          "server_id": "PC-HRD-01",
          "processor_model": "Intel Core i7-10700",
          "processor_cores": 8,
          "processor_clock_ghz": 2.9,
          "disk_type": "SSD",
          "disk_size_gb": 512,
          "disk_health_status": "OK",
          "disk_health_percent": 98,
          "ram_total_gb": 32,
          "ram_slots_used": 2,
          "ram_slots_total": 4,
          "ram_ddr_type": "DDR4",
          "battery_health_percent": 95.5,
          "serial_number": "SGH731S84K",
          "ip_address": "192.168.1.100",
          "mac_address": "00:1A:2B:3C:4D:5E",
          "os_version": "Windows 10 Pro 10.0.19045",
          "updated_at": "2026-10-01T08:38:32Z"
        }
      },
      404: { description: 'Asset info not found', example: { error: "Asset info tidak ditemukan" } },
      500: { description: 'Database error' }
    },
    curl: `curl -X GET http://localhost:8081/api/servers/PC-HRD-01/asset-info \\
  -H "Cookie: session=<session-id>"`,
    tryIt: true
  },
  {
    id: 'get-server-applications',
    tag: 'Dashboard',
    method: 'GET',
    path: '/api/servers/{id}/applications',
    summary: 'Get installed applications for a device',
    description: 'Returns the list of installed applications (from Windows Registry) for a specific device.',
    auth: {
      type: 'session',
      description: 'Requires valid dashboard session',
      headers: ['Cookie: session=<session-id>']
    },
    parameters: [
      { name: 'id', in: 'path', required: true, type: 'string', example: 'PC-HRD-01', description: 'Server ID (e.g., "PC-HRD-01")' }
    ],
    responses: {
      200: {
        description: 'Array of installed applications',
        example: [
          { "app_name": "Google Chrome", "app_version": "118.0.5993.88", "publisher": "Google LLC", "install_date": "2023-10-15", "reported_at": "2026-10-01T08:38:32Z" },
          { "app_name": "Microsoft Office", "app_version": "16.0.15629.20152", "publisher": "Microsoft Corporation", "install_date": "2023-01-15", "reported_at": "2026-10-01T08:38:32Z" }
        ]
      },
      404: { description: 'Server not found', example: { error: "Server tidak ditemukan" } },
      500: { description: 'Database error' }
    },
    curl: `curl -X GET http://localhost:8081/api/servers/PC-HRD-01/applications \\
  -H "Cookie: session=<session-id>"`,
    tryIt: true
  },
  {
    id: 'get-server-users',
    tag: 'Dashboard',
    method: 'GET',
    path: '/api/servers/{id}/users',
    summary: 'Get local user accounts for a device',
    description: 'Returns local user accounts with privilege levels (administrator/standard_user/limited), disabled status, and current login status.',
    auth: {
      type: 'session',
      description: 'Requires valid dashboard session',
      headers: ['Cookie: session=<session-id>']
    },
    parameters: [
      { name: 'id', in: 'path', required: true, type: 'string', example: 'PC-HRD-01', description: 'Server ID (e.g., "PC-HRD-01")' }
    ],
    responses: {
      200: {
        description: 'Array of local user accounts',
        example: [
          { "username": "IT 10", "full_name": "", "privilege": "administrator", "is_disabled": false, "is_logged_in": true, "reported_at": "2026-10-01T08:38:32Z" },
          { "username": "Guest", "full_name": "Guest Account", "privilege": "limited", "is_disabled": true, "is_logged_in": false, "reported_at": "2026-10-01T08:38:32Z" }
        ]
      },
      404: { description: 'Server not found', example: { error: "Server tidak ditemukan" } },
      500: { description: 'Database error' }
    },
    curl: `curl -X GET http://localhost:8081/api/servers/PC-HRD-01/users \\
  -H "Cookie: session=<session-id>"`,
    tryIt: true
  },
  {
    id: 'put-server-metadata',
    tag: 'Dashboard',
    method: 'PUT',
    path: '/api/servers/{id}/metadata',
    summary: 'Update device metadata (owner, department, location, notes)',
    description: 'Updates editable device metadata. All fields are optional; only provided fields are updated. Device type must be one of: server, pc, laptop.',
    auth: {
      type: 'session',
      description: 'Requires valid dashboard session',
      headers: ['Cookie: session=<session-id>', 'Content-Type: application/json']
    },
    parameters: [
      { name: 'id', in: 'path', required: true, type: 'string', example: 'PC-HRD-01', description: 'Server ID (e.g., "PC-HRD-01")' }
    ],
    requestBody: {
      required: false,
      contentType: 'application/json',
      schema: {
        type: 'object',
        properties: {
          device_type: { type: 'string', enum: ['server', 'pc', 'laptop'], description: 'Device type' },
          location: { type: 'string', maxLength: 100, description: 'Physical location (room, building)' },
          owner_name: { type: 'string', maxLength: 100, description: 'Assigned owner name' },
          department: { type: 'string', maxLength: 100, description: 'Department/division' },
          notes: { type: 'string', maxLength: 255, description: 'Additional notes' }
        }
      },
      example: {
        "device_type": "pc",
        "location": "Ruang IT",
        "owner_name": "Bagues IT",
        "department": "IT",
        "notes": "Primary workstation"
      }
    },
    responses: {
      200: { description: 'Metadata updated', example: { status: "updated" } },
      400: { description: 'Validation error (length, invalid device_type)', example: { error: "device_type harus salah satu dari: server, pc, laptop" } },
      404: { description: 'Server not found', example: { error: "Perangkat tidak ditemukan" } },
      500: { description: 'Database error' }
    },
    curl: `curl -X PUT http://localhost:8081/api/servers/PC-HRD-01/metadata \\
  -H "Content-Type: application/json" \\
  -H "Cookie: session=<session-id>" \\
  -d '{
    "device_type": "pc",
    "location": "Ruang IT",
    "owner_name": "Bagues IT",
    "department": "IT",
    "notes": "Primary workstation"
  }'`,
    tryIt: true
  },
  {
    id: 'put-server-rename',
    tag: 'Dashboard',
    method: 'PUT',
    path: '/api/servers/{id}/rename',
    summary: 'Rename device (server_id) with cascade to all historical data',
    description: 'Renames the device server_id. Cascades to all child tables (metrics, service_status, asset_info, installed_applications, device_users) via ON UPDATE CASCADE. New ID must be unique and follow naming rules.',
    auth: {
      type: 'session',
      description: 'Requires valid dashboard session',
      headers: ['Cookie: session=<session-id>', 'Content-Type: application/json']
    },
    parameters: [
      { name: 'id', in: 'path', required: true, type: 'string', description: 'Current Server ID (e.g., "PC-HRD-01")' }
    ],
    requestBody: {
      required: true,
      contentType: 'application/json',
      schema: {
        type: 'object',
        required: ['new_server_id'],
        properties: {
          new_server_id: { type: 'string', pattern: '^[a-zA-Z0-9._-]{1,50}$', description: 'New unique server ID (alphanumeric, dot, underscore, hyphen only, max 50 chars)' }
        }
      },
      example: { "new_server_id": "PC-HRD-02" }
    },
    responses: {
      200: { description: 'Device renamed successfully', example: { "status": "renamed", "old_server_id": "PC-HRD-01", "new_server_id": "PC-HRD-02" } },
      400: { description: 'Invalid new_server_id (format, length, or same as current)', example: { error: "new_server_id hanya boleh berisi huruf, angka, titik, underscore, dan tanda hubung" } },
      404: { description: 'Source server not found', example: { error: "Perangkat tidak ditemukan" } },
      409: { description: 'New server_id already exists', example: { error: "Nama 'PC-HRD-02' sudah dipakai perangkat lain" } },
      500: { description: 'Database error' }
    },
    curl: `curl -X PUT http://localhost:8081/api/servers/PC-HRD-01/rename \\
  -H "Content-Type: application/json" \\
  -H "Cookie: session=<session-id>" \\
  -d '{ "new_server_id": "PC-HRD-02" }'`,
    tryIt: true
  },
  {
    id: 'get-upgrade-insights',
    tag: 'Dashboard',
    method: 'GET',
    path: '/api/upgrade-insights',
    summary: 'Evaluate hardware upgrade readiness across devices',
    description: 'Aggregates CPU/RAM/Disk metrics over the selected day window (default 30 days) and hardware inventory (RAM slots, DDR, MHz, disk type, SMART, battery, BIOS age) to recommend targeted upgrades.',
    auth: { type: 'none', description: 'Read-only analytics endpoint' },
    parameters: [
      { name: 'days', in: 'query', required: false, type: 'integer', default: 30, description: 'Lookback window in days (1-365)' }
    ],
    responses: {
      200: { description: 'Upgrade readiness summary and device recommendations', example: { window_days: 30, total_devices: 1, critical_count: 1, devices: [] } }
    },
    curl: `curl "http://localhost:8081/api/upgrade-insights?days=30"`,
    tryIt: true
  },
  {
    id: 'get-policy-audit',
    tag: 'Dashboard',
    method: 'GET',
    path: '/api/policy-audit',
    summary: 'Audit software policy, licensing, and endpoint security',
    description: 'Evaluates all devices against software_policies (installed apps & running processes), Windows/Office activation channels, Antivirus/Firewall state, and local admin accounts.',
    auth: { type: 'none', description: 'Read-only policy audit endpoint' },
    responses: {
      200: { description: 'Compliance summary and per-device audit findings', example: { total_devices: 1, violation_devices: 0, devices: [] } }
    },
    curl: `curl http://localhost:8081/api/policy-audit`,
    tryIt: true
  },
  {
    id: 'get-policies',
    tag: 'Dashboard',
    method: 'GET',
    path: '/api/policies',
    summary: 'List company software policy rules',
    description: 'Returns all configured forbidden, required, and licensed software policy patterns.',
    auth: { type: 'none', description: 'Read-only policy list' },
    responses: {
      200: { description: 'Array of software policy rules', example: [] }
    },
    curl: `curl http://localhost:8081/api/policies`,
    tryIt: true
  },
  {
    id: 'post-policies',
    tag: 'DashboardWrite',
    method: 'POST',
    path: '/api/policies',
    summary: 'Create a software policy rule',
    description: 'Adds a new forbidden, required, or licensed software pattern for compliance auditing.',
    auth: { type: 'session', description: 'Dashboard management endpoint', headers: ['Content-Type: application/json'] },
    requestBody: {
      required: true,
      contentType: 'application/json',
      schema: {
        type: 'object',
        required: ['pattern', 'policy_type'],
        properties: {
          pattern: { type: 'string' },
          policy_type: { type: 'string', enum: ['forbidden', 'required', 'licensed'] },
          category: { type: 'string' },
          licensed_seats: { type: 'integer' },
          notes: { type: 'string' }
        }
      },
      example: { pattern: 'AnyDesk', policy_type: 'forbidden', category: 'Remote Access', notes: 'Gunakan remote tool resmi IT' }
    },
    responses: {
      201: { description: 'Policy rule created' },
      400: { description: 'Invalid pattern or policy_type' },
      409: { description: 'Pattern already exists' }
    },
    curl: `curl -X POST http://localhost:8081/api/policies -H "Content-Type: application/json" -d '{"pattern":"AnyDesk","policy_type":"forbidden"}'`,
    tryIt: true
  },
  {
    id: 'delete-policy',
    tag: 'DashboardWrite',
    method: 'DELETE',
    path: '/api/policies/{id}',
    summary: 'Delete a software policy rule',
    description: 'Deletes a software policy rule by ID.',
    auth: { type: 'session', description: 'Dashboard management endpoint' },
    parameters: [
      { name: 'id', in: 'path', required: true, type: 'integer', description: 'Policy ID' }
    ],
    responses: {
      200: { description: 'Deleted', example: { status: 'deleted' } }
    },
    curl: `curl -X DELETE http://localhost:8081/api/policies/1`,
    tryIt: true
  }
];

// Tag definitions for sidebar grouping
export const apiTags = [
  { id: 'Agent', label: '📡 Agent Endpoints', description: 'Endpoints called by agents to send data', endpoints: ['post-metrics', 'post-asset-info'] },
  { id: 'Dashboard', label: '📖 Dashboard GET Endpoints', description: 'Read-only endpoints for dashboard consumption', endpoints: ['get-servers', 'get-server-metrics', 'get-server-status', 'get-server-asset-info', 'get-server-applications', 'get-server-users', 'get-upgrade-insights', 'get-policy-audit', 'get-policies'] },
  { id: 'DashboardWrite', label: '✏️ Dashboard Write Endpoints', description: 'Write operations for device & policy management', endpoints: ['put-server-metadata', 'put-server-rename', 'post-policies', 'delete-policy'] },
  { id: 'AuthErrors', label: '🔐 Authentication & Errors', description: 'Authentication methods and error code reference', endpoints: [] }
];

// Environment configurations
export const environments = [
  { id: 'development', name: 'Development', baseUrl: 'http://localhost:8081', color: 'text-forest-moss' },
  { id: 'staging', name: 'Staging', baseUrl: 'https://staging-monitoring.company.com', color: 'text-gold-dim' },
  { id: 'production', name: 'Production', baseUrl: 'https://monitoring.company.com', color: 'text-cardinal' }
];

// Helper to get endpoint by id
export function getEndpointById(id) {
  return apiEndpoints.find(e => e.id === id);
}

// Helper to get endpoints by tag
export function getEndpointsByTag(tagId) {
  const tag = apiTags.find(t => t.id === tagId);
  if (!tag) return [];
  return tag.endpoints.map(id => getEndpointById(id)).filter(Boolean);
}

// Build OpenAPI 3.0 document from endpoint data
export function buildOpenApiSpec() {
  const paths = {};
  apiEndpoints.forEach(ep => {
    const operation = {
      tags: [ep.tag],
      summary: ep.summary,
      description: ep.description,
      parameters: (ep.parameters || []).map(p => ({
        name: p.name,
        in: p.in,
        required: p.required,
        description: p.description,
        schema: {
          type: p.type,
          ...(p.default !== undefined ? { default: p.default } : {}),
          ...(p.enum ? { enum: p.enum } : {}),
          ...(p.pattern ? { pattern: p.pattern } : {})
        }
      })),
      responses: {}
    };
    if (ep.requestBody) {
      operation.requestBody = {
        required: ep.requestBody.required,
        content: {
          [ep.requestBody.contentType || 'application/json']: {
            schema: ep.requestBody.schema,
            ...(ep.requestBody.example ? { example: ep.requestBody.example } : {})
          }
        }
      };
    }
    Object.entries(ep.responses).forEach(([code, resp]) => {
      operation.responses[code] = {
        description: resp.description,
        ...(resp.example !== undefined ? { content: { 'application/json': { example: resp.example } } } : {})
      };
    });
    if (!paths[ep.path]) paths[ep.path] = {};
    paths[ep.path][ep.method.toLowerCase()] = operation;
  });

  return {
    openapi: '3.0.3',
    info: {
      title: 'Seneschal Collector API',
      version: '2.0.0',
      description: 'REST API for the Seneschal infrastructure monitoring system: agent telemetry ingestion and dashboard read/write endpoints.'
    },
    servers: environments.map(e => ({ url: e.baseUrl, description: e.name })),
    tags: apiTags.filter(t => t.id !== 'AuthErrors').map(t => ({ name: t.id, description: t.description })),
    paths
  };
}