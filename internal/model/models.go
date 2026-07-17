package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"uniqueIndex;not null;size:64" json:"username"`
	Password  string         `gorm:"not null" json:"-"` // bcrypt hash, never serialize
	Role      string         `gorm:"default:'user';size:20" json:"role"` // admin / user
	Active    bool           `gorm:"default:true" json:"active"`
	Timezone  string         `gorm:"size:50" json:"timezone"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type Monitor struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            uint           `gorm:"index;not null" json:"user_id"`
	Name              string         `gorm:"not null;size:150" json:"name"`
	Description       string         `gorm:"size:500" json:"description"`
	Type              string         `gorm:"not null;size:20" json:"type"` // http, tcp, ping, dns
	Active            bool           `gorm:"default:true" json:"active"`
	URL               string         `gorm:"size:2000" json:"url"`
	Hostname          string         `gorm:"size:255" json:"hostname"`
	Port              uint16         `gorm:"default:0" json:"port"`
	Method            string         `gorm:"default:'GET';size:10" json:"method"`
	Interval          uint32         `gorm:"default:60" json:"interval"` // seconds, min 3
	Timeout           float64        `gorm:"default:30" json:"timeout"`
	MaxRetries        uint32         `gorm:"default:0" json:"max_retries"`
	RetryInterval     uint32         `gorm:"default:0" json:"retry_interval"`
	ResendInterval    uint32         `gorm:"default:0" json:"resend_interval"`

	Headers            string         `gorm:"type:text" json:"headers"`              // JSON string
	Body               string         `gorm:"type:text" json:"body"`
	AcceptedStatusCodes string        `gorm:"type:text;default:'[\"200-299\"]'" json:"accepted_status_codes"`
	Keyword            string         `gorm:"size:255" json:"keyword"`
	InvertKeyword      bool           `gorm:"default:false" json:"invert_keyword"`
	IgnoreTLS          bool           `gorm:"default:false" json:"ignore_tls"`
	UpsideDown         bool           `gorm:"default:false" json:"upside_down"`
	MaxRedirects       uint32         `gorm:"default:10" json:"max_redirects"`

	AuthMethod     string `gorm:"size:20" json:"auth_method"` // basic, bearer, oauth2-cc, ntlm, mtls
	BasicAuthUser  string `gorm:"size:255" json:"basic_auth_user"`
	BasicAuthPass  string `gorm:"size:255" json:"-"` // sensitive
	BearerToken    string `gorm:"size:2000" json:"-"`
	AuthWorkstation string `gorm:"size:255" json:"auth_workstation"`
	AuthDomain     string `gorm:"size:255" json:"auth_domain"`

	TLSKey  string `gorm:"type:text" json:"-"`
	TLSCert string `gorm:"type:text" json:"-"`
	TLSCa   string `gorm:"type:text" json:"tls_ca"`

	OAuthClientID     string `gorm:"type:text" json:"-"`
	OAuthClientSecret string `gorm:"type:text" json:"-"`
	OAuthTokenURL     string `gorm:"type:text" json:"oauth_token_url"`
	OAuthScopes       string `gorm:"type:text" json:"oauth_scopes"`
	OAuthAuthMethod   string `gorm:"type:text" json:"oauth_auth_method"`
	OAuthAudience     string `gorm:"type:text" json:"oauth_audience"`

	DNSResolveType   string `gorm:"size:5" json:"dns_resolve_type"`
	DNSResolveServer string `gorm:"size:255" json:"dns_resolve_server"`
	DNSLastResult    string `gorm:"size:255" json:"dns_last_result"`

	PushToken             string `gorm:"size:32" json:"push_token"`
	PacketSize            uint32 `gorm:"default:56" json:"packet_size"`
	ExpiryNotification    bool   `gorm:"default:true" json:"expiry_notification"`
	HTTPBodyEncoding      string `gorm:"size:25;default:'json'" json:"http_body_encoding"`
	RetryOnlyOnStatusCode bool   `gorm:"default:false" json:"retry_only_on_status_code"`
	CacheBust             bool   `gorm:"default:false" json:"cache_bust"`
	SaveResponse          bool   `gorm:"default:false" json:"save_response"`
	SaveErrorResponse     bool   `gorm:"default:false" json:"save_error_response"`
	ResponseMaxLength     uint32 `gorm:"default:4096" json:"response_max_length"`

	PingNumeric           bool   `gorm:"default:false" json:"ping_numeric"`
	PingCount             uint32 `gorm:"default:4" json:"ping_count"`
	PingPerRequestTimeout uint32 `gorm:"default:1000" json:"ping_per_request_timeout"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Heartbeat struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MonitorID uint      `gorm:"index;not null" json:"monitor_id"`
	Status    uint16    `gorm:"not null" json:"status"` // 0=DOWN 1=UP 2=PENDING
	Msg       string    `gorm:"type:text" json:"msg"`
	PingMS    *float64  `json:"ping_ms"`
	HTTPStatus int16    `gorm:"default:0" json:"http_status"`
	Important bool      `gorm:"default:false" json:"important"`
	Retries   uint32    `gorm:"default:0" json:"retries"`
	DownCount uint32    `gorm:"default:0" json:"down_count"`
	Time      time.Time `gorm:"not null;index" json:"time"`
	EndTime   time.Time `json:"end_time"`
	Duration  uint32    `gorm:"default:0" json:"duration"`
}

type StatMinutely struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	MonitorID uint    `gorm:"uniqueIndex:idx_monitor_time_min;not null" json:"monitor_id"`
	Timestamp int64   `gorm:"uniqueIndex:idx_monitor_time_min;not null" json:"timestamp"`
	Up        uint32  `gorm:"default:0" json:"up"`
	Down      uint32  `gorm:"default:0" json:"down"`
	AvgPing   float64 `gorm:"type:numeric(10,2);default:0" json:"avg_ping"`
	MinPing   float64 `gorm:"type:numeric(10,2);default:0" json:"min_ping"`
	MaxPing   float64 `gorm:"type:numeric(10,2);default:0" json:"max_ping"`
}

type StatHourly struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	MonitorID uint    `gorm:"uniqueIndex:idx_monitor_time_hour;not null" json:"monitor_id"`
	Timestamp int64   `gorm:"uniqueIndex:idx_monitor_time_hour;not null" json:"timestamp"`
	Up        uint32  `gorm:"default:0" json:"up"`
	Down      uint32  `gorm:"default:0" json:"down"`
	AvgPing   float64 `gorm:"type:numeric(10,2);default:0" json:"avg_ping"`
	MinPing   float64 `gorm:"type:numeric(10,2);default:0" json:"min_ping"`
	MaxPing   float64 `gorm:"type:numeric(10,2);default:0" json:"max_ping"`
}

type StatDaily struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	MonitorID uint    `gorm:"uniqueIndex:idx_monitor_time_day;not null" json:"monitor_id"`
	Timestamp int64   `gorm:"uniqueIndex:idx_monitor_time_day;not null" json:"timestamp"`
	Up        uint32  `gorm:"default:0" json:"up"`
	Down      uint32  `gorm:"default:0" json:"down"`
	AvgPing   float64 `gorm:"type:numeric(10,2);default:0" json:"avg_ping"`
	MinPing   float64 `gorm:"type:numeric(10,2);default:0" json:"min_ping"`
	MaxPing   float64 `gorm:"type:numeric(10,2);default:0" json:"max_ping"`
}

type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Name      string    `gorm:"not null;size:255" json:"name"`
	Type      string    `gorm:"not null;size:20" json:"type"` // feishu, email
	Config    string    `gorm:"type:text" json:"config"`      // JSON
	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MonitorNotification struct {
	ID             uint `gorm:"primaryKey" json:"id"`
	MonitorID      uint `gorm:"uniqueIndex:idx_monitor_notif;not null" json:"monitor_id"`
	NotificationID uint `gorm:"uniqueIndex:idx_monitor_notif;not null" json:"notification_id"`
}

type Tag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null;size:255" json:"name"`
	Color     string    `gorm:"not null;size:255" json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

type MonitorTag struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	MonitorID uint   `gorm:"index;not null" json:"monitor_id"`
	TagID     uint   `gorm:"index;not null" json:"tag_id"`
	Value     string `gorm:"size:255" json:"value"`
}

type Incident struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	MonitorID    uint      `gorm:"index;not null" json:"monitor_id"`
	Title        string    `gorm:"not null;size:255" json:"title"`
	Status       uint16    `gorm:"default:0" json:"status"` // 0=DOWN 1=UP
	StartedAt    time.Time `gorm:"not null" json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
	DurationSec  uint32    `gorm:"default:0" json:"duration_seconds"`
	Msg          string    `gorm:"type:text" json:"msg"`
	CreatedAt    time.Time `json:"created_at"`
}

type SLAReport struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	PeriodType  string    `gorm:"not null;size:20" json:"period_type"` // day, week, month, quarter, year
	PeriodStart time.Time `gorm:"not null" json:"period_start"`
	PeriodEnd   time.Time `gorm:"not null" json:"period_end"`
	DataJSON    string    `gorm:"type:text" json:"-"` // JSON blob
	GeneratedAt time.Time `json:"generated_at"`
}

type Setting struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Key   string `gorm:"uniqueIndex;not null;size:200" json:"key"`
	Value string `gorm:"type:text" json:"value"`
	Type  string `gorm:"size:20" json:"type"`
}