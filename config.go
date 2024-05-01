package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/nknorg/nkn-sdk-go"
)

// Config represents the configuration data
type Config struct {
	Seed        string   `json:"seed"`
	Title       string   `json:"title"`
	Owner       string   `json:"owner"`
	Transcoders []string `json:transcoders`
}

type Transcode struct {
	Resolution int
	Framerate  int
}

// NewConfig reads the configuration file from a specified location and populates defaults
func NewConfig(configFile string) (*Config, error) {

	// Generate mediaMTX config if doesnt exist;
	generateMediaMTXConfig()

	// Check if the file exists
	_, err := os.Stat(configFile)
	if err != nil {
		if os.IsNotExist(err) {

			//generate a wallet
			newWallet, _ := nkn.NewAccount(nil)
			seedStr := hex.EncodeToString(newWallet.Seed())
			// Create a default configuration
			defaultConfig := &Config{Seed: seedStr, Title: "Unnamed Stream"}
			data, err := json.MarshalIndent(defaultConfig, "", "  ")
			if err != nil {
				return nil, err
			}
			err = os.WriteFile(configFile, data, 0644) // Set appropriate permissions
			if err != nil {
				return nil, fmt.Errorf("error creating config file: %w", err)
			}
			return defaultConfig, nil
		}
		return nil, err
	}

	// Open the existing file and decode data
	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}

	// Set defaults for missing fields
	if cfg.Title == "" {
		cfg.Title = "Unnamed Stream"
	}

	return &cfg, nil
}

func getTranscoders(config *Config) []Transcode {
	var transcoders = make([]Transcode, 0)

	for _, v := range config.Transcoders {
		transcodeStr := strings.Split(v, "p")
		resolution, err := strconv.Atoi(transcodeStr[0])
		if err != nil {
			fmt.Println("Skipping invalid transcode value in config:", v)
			continue
		}

		if sourceResolution <= resolution {
			fmt.Println("Skipping transcode value in config:", v, "stream source is smaller:", sourceResolution)
			continue
		}

		framerate := 30
		if len(transcodeStr) == 2 && len(transcodeStr[1]) > 0 {
			framerate, err = strconv.Atoi(transcodeStr[1])
			if err != nil {
				fmt.Println("Skipping invalid transcode value in config:", v)
				continue
			}
		}

		if framerate > sourceFramerate {
			framerate = sourceFramerate
			fmt.Println("Lowering transcode framerate value in config:", v, "stream source framerate:", sourceFramerate)
		}

		if sourceResolution == resolution && framerate == sourceFramerate {
			fmt.Println("Skipping transcode value in config:", v, "stream source resolution and framerate are equal")
			continue
		}

		transcoders = append(transcoders, Transcode{
			Resolution: resolution,
			Framerate:  framerate,
		})
	}

	return removeDuplicateTranscodes(transcoders)
}

func removeDuplicateTranscodes(transcodes []Transcode) []Transcode {
	// Sort the unique slice by resolution (descending) and then framerate (descending)
	sort.SliceStable(transcodes, func(i, j int) bool {
		if transcodes[i].Resolution != transcodes[j].Resolution {
			return transcodes[i].Resolution > transcodes[j].Resolution // Descending order for resolution
		}
		return transcodes[i].Framerate > transcodes[j].Framerate // Descending order for framerate
	})

	// Create a map to store seen resolutions and their corresponding framerates
	seen := make(map[int]int)
	var unique []Transcode

	// Loop through the original slice
	for _, transcode := range transcodes {
		// Check if the resolution is already seen
		if prevFramerate, ok := seen[transcode.Resolution]; !ok || transcode.Framerate > prevFramerate {
			// If not seen or framerate is higher, update seen map and add to unique slice
			seen[transcode.Resolution] = transcode.Framerate
			unique = append(unique, transcode)
		}
	}

	return unique
}

func generateMediaMTXConfig() {
	if _, err := os.Stat("mediamtx.yml"); errors.Is(err, os.ErrNotExist) {
		os.WriteFile("mediamtx.yml", []byte(mediaMTXDefaults), 0644)
	}
}

const mediaMTXDefaults = `###############################################
# Global settings

# Settings in this section are applied anywhere.

###############################################
# Global settings -> General

# Verbosity of the program; available values are "error", "warn", "info", "debug".
logLevel: info
# Destinations of log messages; available values are "stdout", "file" and "syslog".
logDestinations: [stdout]
# If "file" is in logDestinations, this is the file which will receive the logs.
logFile: mediamtx.log

# Timeout of read operations.
readTimeout: 10s
# Timeout of write operations.
writeTimeout: 10s
# Size of the queue of outgoing packets.
# A higher value allows to increase throughput, a lower value allows to save RAM.
writeQueueSize: 512
# Maximum size of outgoing UDP packets.
# This can be decreased to avoid fragmentation on networks with a low UDP MTU.
udpMaxPayloadSize: 1472

# Enable Prometheus-compatible metrics.
metrics: no
# Address of the metrics listener.
metricsAddress: :9998

# Enable pprof-compatible endpoint to monitor performances.
pprof: no
# Address of the pprof listener.
pprofAddress: :9999

# Command to run when a client connects to the server.
# This is terminated with SIGINT when a client disconnects from the server.
# The following environment variables are available:
# * RTSP_PORT: RTSP server port
# * MTX_CONN_TYPE: connection type
# * MTX_CONN_ID: connection ID
runOnConnect:
# Restart the command if it exits.
runOnConnectRestart: no
# Command to run when a client disconnects from the server.
# Environment variables are the same of runOnConnect.
runOnDisconnect:

###############################################
# Global settings -> Authentication

# Authentication method. Available values are:
# * internal: users are stored in the configuration file
# * http: an external HTTP URL is contacted to perform authentication
# * jwt: an external identity server provides authentication through JWTs
authMethod: internal

# Internal authentication.
# list of users.
authInternalUsers:
  # Default unprivileged user.
  # Username. 'any' means any user, including anonymous ones.
- user: any
  # Password. Not used in case of 'any' user.
  pass:
  # IPs or networks allowed to use this user. An empty list means any IP.
  ips: []
  # List of permissions.
  permissions:
    # Available actions are: publish, read, playback, api, metrics, pprof.
  - action: publish
    # Paths can be set to further restrict access to a specific path.
    # An empty path means any path.
    # Regular expressions can be used by using a tilde as prefix.
    path:
  - action: read
    path:
  - action: playback
    path:

  # Default administrator.
  # This allows to use API, metrics and PPROF without authentication,
  # if the IP is localhost.
- user: any
  pass:
  ips: ['127.0.0.1', '::1']
  permissions:
  - action: api
  - action: metrics
  - action: pprof

# HTTP-based authentication.
# URL called to perform authentication. Every time a user wants
# to authenticate, the server calls this URL with the POST method
# and a body containing:
# {
#   "user": "user",
#   "password": "password",
#   "ip": "ip",
#   "action": "publish|read|playback|api|metrics|pprof",
#   "path": "path",
#   "protocol": "rtsp|rtmp|hls|webrtc|srt",
#   "id": "id",
#   "query": "query"
# }
# If the response code is 20x, authentication is accepted, otherwise
# it is discarded.
authHTTPAddress:
# Actions to exclude from HTTP-based authentication.
# Format is the same as the one of user permissions.
authHTTPExclude:
- action: api
- action: metrics
- action: pprof

# JWT-based authentication.
# Users have to login through an external identity server and obtain a JWT.
# This JWT must contain the claim "mediamtx_permissions" with permissions,
# for instance:
# {
#  ...
#  "mediamtx_permissions": [
#     {
#       "action": "publish",
#       "path": "somepath"
#     }
#   ]
# }
# Users are then expected to pass the JWT as a query parameter, i.e. ?jwt=...
# This is the JWKS URL that will be used to pull (once) the public key that allows
# to validate JWTs.
authJWTJWKS:

###############################################
# Global settings -> API

# Enable controlling the server through the API.
api: no
# Address of the API listener.
apiAddress: :9997

###############################################
# Global settings -> Playback server

# Enable downloading recordings from the playback server.
playback: no
# Address of the playback server listener.
playbackAddress: :9996

###############################################
# Global settings -> RTSP server

# Enable publishing and reading streams with the RTSP protocol.
rtsp: no
# List of enabled RTSP transport protocols.
# UDP is the most performant, but doesn't work when there's a NAT/firewall between
# server and clients, and doesn't support encryption.
# UDP-multicast allows to save bandwidth when clients are all in the same LAN.
# TCP is the most versatile, and does support encryption.
# The handshake is always performed with TCP.
protocols: [udp, multicast, tcp]
# Encrypt handshakes and TCP streams with TLS (RTSPS).
# Available values are "no", "strict", "optional".
encryption: "no"
# Address of the TCP/RTSP listener. This is needed only when encryption is "no" or "optional".
rtspAddress: :8554
# Address of the TCP/TLS/RTSPS listener. This is needed only when encryption is "strict" or "optional".
rtspsAddress: :8322
# Address of the UDP/RTP listener. This is needed only when "udp" is in protocols.
rtpAddress: :8000
# Address of the UDP/RTCP listener. This is needed only when "udp" is in protocols.
rtcpAddress: :8001
# IP range of all UDP-multicast listeners. This is needed only when "multicast" is in protocols.
multicastIPRange: 224.1.0.0/16
# Port of all UDP-multicast/RTP listeners. This is needed only when "multicast" is in protocols.
multicastRTPPort: 8002
# Port of all UDP-multicast/RTCP listeners. This is needed only when "multicast" is in protocols.
multicastRTCPPort: 8003
# Path to the server key. This is needed only when encryption is "strict" or "optional".
# This can be generated with:
# openssl genrsa -out server.key 2048
# openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650
serverKey: server.key
# Path to the server certificate. This is needed only when encryption is "strict" or "optional".
serverCert: server.crt
# Authentication methods. Available are "basic" and "digest".
# "digest" doesn't provide any additional security and is available for compatibility only.
rtspAuthMethods: [basic]

###############################################
# Global settings -> RTMP server

# Enable publishing and reading streams with the RTMP protocol.
rtmp: yes
# Address of the RTMP listener. This is needed only when encryption is "no" or "optional".
rtmpAddress: :1935
# Encrypt connections with TLS (RTMPS).
# Available values are "no", "strict", "optional".
rtmpEncryption: "no"
# Address of the RTMPS listener. This is needed only when encryption is "strict" or "optional".
rtmpsAddress: :1936
# Path to the server key. This is needed only when encryption is "strict" or "optional".
# This can be generated with:
# openssl genrsa -out server.key 2048
# openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650
rtmpServerKey: server.key
# Path to the server certificate. This is needed only when encryption is "strict" or "optional".
rtmpServerCert: server.crt

###############################################
# Global settings -> HLS server

# Enable reading streams with the HLS protocol.
hls: yes
# Address of the HLS listener.
hlsAddress: :8888
# Enable TLS/HTTPS on the HLS server.
# This is required for Low-Latency HLS.
hlsEncryption: no
# Path to the server key. This is needed only when encryption is yes.
# This can be generated with:
# openssl genrsa -out server.key 2048
# openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650
hlsServerKey: server.key
# Path to the server certificate.
hlsServerCert: server.crt
# By default, HLS is generated only when requested by a user.
# This option allows to generate it always, avoiding the delay between request and generation.
hlsAlwaysRemux: yes
# Variant of the HLS protocol to use. Available options are:
# * mpegts - uses MPEG-TS segments, for maximum compatibility.
# * fmp4 - uses fragmented MP4 segments, more efficient.
# * lowLatency - uses Low-Latency HLS.
hlsVariant: mpegts
# Number of HLS segments to keep on the server.
# Segments allow to seek through the stream.
# Their number doesn't influence latency.
hlsSegmentCount: 0
# Minimum duration of each segment.
# A player usually puts 3 segments in a buffer before reproducing the stream.
# The final segment duration is also influenced by the interval between IDR frames,
# since the server changes the duration in order to include at least one IDR frame
# in each segment.
hlsSegmentDuration: 1s
# Minimum duration of each part.
# A player usually puts 3 parts in a buffer before reproducing the stream.
# Parts are used in Low-Latency HLS in place of segments.
# Part duration is influenced by the distance between video/audio samples
# and is adjusted in order to produce segments with a similar duration.
hlsPartDuration: 200ms
# Maximum size of each segment.
# This prevents RAM exhaustion.
hlsSegmentMaxSize: 50M
# Value of the Access-Control-Allow-Origin header provided in every HTTP response.
# This allows to play the HLS stream from an external website.
hlsAllowOrigin: '*'
# List of IPs or CIDRs of proxies placed before the HLS server.
# If the server receives a request from one of these entries, IP in logs
# will be taken from the X-Forwarded-For header.
hlsTrustedProxies: []
# Directory in which to save segments, instead of keeping them in the RAM.
# This decreases performance, since reading from disk is less performant than
# reading from RAM, but allows to save RAM.
hlsDirectory: ''

###############################################
# Global settings -> WebRTC server

# Enable publishing and reading streams with the WebRTC protocol.
webrtc: no
# Address of the WebRTC HTTP listener.
webrtcAddress: :8889
# Enable TLS/HTTPS on the WebRTC server.
webrtcEncryption: no
# Path to the server key.
# This can be generated with:
# openssl genrsa -out server.key 2048
# openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650
webrtcServerKey: server.key
# Path to the server certificate.
webrtcServerCert: server.crt
# Value of the Access-Control-Allow-Origin header provided in every HTTP response.
# This allows to play the WebRTC stream from an external website.
webrtcAllowOrigin: '*'
# List of IPs or CIDRs of proxies placed before the WebRTC server.
# If the server receives a request from one of these entries, IP in logs
# will be taken from the X-Forwarded-For header.
webrtcTrustedProxies: []
# Address of a local UDP listener that will receive connections.
# Use a blank string to disable.
webrtcLocalUDPAddress: :8189
# Address of a local TCP listener that will receive connections.
# This is disabled by default since TCP is less efficient than UDP and
# introduces a progressive delay when network is congested.
webrtcLocalTCPAddress: ''
# WebRTC clients need to know the IP of the server.
# Gather IPs from interfaces and send them to clients.
webrtcIPsFromInterfaces: yes
# List of interfaces whose IPs will be sent to clients.
# An empty value means to use all available interfaces.
webrtcIPsFromInterfacesList: []
# List of additional hosts or IPs to send to clients.
webrtcAdditionalHosts: []
# ICE servers. Needed only when local listeners can't be reached by clients.
# STUN servers allows to obtain and share the public IP of the server.
# TURN/TURNS servers forces all traffic through them.
webrtcICEServers2: []
  # - url: stun:stun.l.google.com:19302
  # if user is "AUTH_SECRET", then authentication is secret based.
  # the secret must be inserted into the password field.
  # username: ''
  # password: ''
  # clientOnly: false

###############################################
# Global settings -> SRT server

# Enable publishing and reading streams with the SRT protocol.
srt: no
# Address of the SRT listener.
srtAddress: :8890

###############################################
# Default path settings

# Settings in "pathDefaults" are applied anywhere,
# unless they are overridden in "paths".
pathDefaults:

  ###############################################
  # Default path settings -> General

  # Source of the stream. This can be:
  # * publisher -> the stream is provided by a RTSP, RTMP, WebRTC or SRT client
  # * rtsp://existing-url -> the stream is pulled from another RTSP server / camera
  # * rtsps://existing-url -> the stream is pulled from another RTSP server / camera with RTSPS
  # * rtmp://existing-url -> the stream is pulled from another RTMP server / camera
  # * rtmps://existing-url -> the stream is pulled from another RTMP server / camera with RTMPS
  # * http://existing-url/stream.m3u8 -> the stream is pulled from another HLS server / camera
  # * https://existing-url/stream.m3u8 -> the stream is pulled from another HLS server / camera with HTTPS
  # * udp://ip:port -> the stream is pulled with UDP, by listening on the specified IP and port
  # * srt://existing-url -> the stream is pulled from another SRT server / camera
  # * whep://existing-url -> the stream is pulled from another WebRTC server / camera
  # * wheps://existing-url -> the stream is pulled from another WebRTC server / camera with HTTPS
  # * redirect -> the stream is provided by another path or server
  # * rpiCamera -> the stream is provided by a Raspberry Pi Camera
  # If path name is a regular expression, $G1, G2, etc will be replaced
  # with regular expression groups.
  source: publisher
  # If the source is a URL, and the source certificate is self-signed
  # or invalid, you can provide the fingerprint of the certificate in order to
  # validate it anyway. It can be obtained by running:
  # openssl s_client -connect source_ip:source_port </dev/null 2>/dev/null | sed -n '/BEGIN/,/END/p' > server.crt
  # openssl x509 -in server.crt -noout -fingerprint -sha256 | cut -d "=" -f2 | tr -d ':'
  sourceFingerprint:
  # If the source is a URL, it will be pulled only when at least
  # one reader is connected, saving bandwidth.
  sourceOnDemand: no
  # If sourceOnDemand is "yes", readers will be put on hold until the source is
  # ready or until this amount of time has passed.
  sourceOnDemandStartTimeout: 10s
  # If sourceOnDemand is "yes", the source will be closed when there are no
  # readers connected and this amount of time has passed.
  sourceOnDemandCloseAfter: 10s
  # Maximum number of readers. Zero means no limit.
  maxReaders: 0
  # SRT encryption passphrase require to read from this path
  srtReadPassphrase:
  # If the stream is not available, redirect readers to this path.
  # It can be can be a relative path (i.e. /otherstream) or an absolute RTSP URL.
  fallback:

  ###############################################
  # Default path settings -> Record

  # Record streams to disk.
  record: no
  # Path of recording segments.
  # Extension is added automatically.
  # Available variables are %path (path name), %Y %m %d %H %M %S %f %s (time in strftime format)
  recordPath: ./recordings/%path/%Y-%m-%d_%H-%M-%S-%f
  # Format of recorded segments.
  # Available formats are "fmp4" (fragmented MP4) and "mpegts" (MPEG-TS).
  recordFormat: fmp4
  # fMP4 segments are concatenation of small MP4 files (parts), each with this duration.
  # MPEG-TS segments are concatenation of 188-bytes packets, flushed to disk with this period.
  # When a system failure occurs, the last part gets lost.
  # Therefore, the part duration is equal to the RPO (recovery point objective).
  recordPartDuration: 100ms
  # Minimum duration of each segment.
  recordSegmentDuration: 1h
  # Delete segments after this timespan.
  # Set to 0s to disable automatic deletion.
  recordDeleteAfter: 24h

  ###############################################
  # Default path settings -> Publisher source (when source is "publisher")

  # Allow another client to disconnect the current publisher and publish in its place.
  overridePublisher: yes
  # SRT encryption passphrase required to publish to this path
  srtPublishPassphrase:

  ###############################################
  # Default path settings -> RTSP source (when source is a RTSP or a RTSPS URL)

  # Transport protocol used to pull the stream. available values are "automatic", "udp", "multicast", "tcp".
  rtspTransport: automatic
  # Support sources that don't provide server ports or use random server ports. This is a security issue
  # and must be used only when interacting with sources that require it.
  rtspAnyPort: no
  # Range header to send to the source, in order to start streaming from the specified offset.
  # available values:
  # * clock: Absolute time
  # * npt: Normal Play Time
  # * smpte: SMPTE timestamps relative to the start of the recording
  rtspRangeType:
  # Available values:
  # * clock: UTC ISO 8601 combined date and time string, e.g. 20230812T120000Z
  # * npt: duration such as "300ms", "1.5m" or "2h45m", valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h"
  # * smpte: duration such as "300ms", "1.5m" or "2h45m", valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h"
  rtspRangeStart:

  ###############################################
  # Default path settings -> Redirect source (when source is "redirect")

  # RTSP URL which clients will be redirected to.
  sourceRedirect:

  ###############################################
  # Default path settings -> Raspberry Pi Camera source (when source is "rpiCamera")

  # ID of the camera
  rpiCameraCamID: 0
  # width of frames
  rpiCameraWidth: 1920
  # height of frames
  rpiCameraHeight: 1080
  # flip horizontally
  rpiCameraHFlip: false
  # flip vertically
  rpiCameraVFlip: false
  # brightness [-1, 1]
  rpiCameraBrightness: 0
  # contrast [0, 16]
  rpiCameraContrast: 1
  # saturation [0, 16]
  rpiCameraSaturation: 1
  # sharpness [0, 16]
  rpiCameraSharpness: 1
  # exposure mode.
  # values: normal, short, long, custom
  rpiCameraExposure: normal
  # auto-white-balance mode.
  # values: auto, incandescent, tungsten, fluorescent, indoor, daylight, cloudy, custom
  rpiCameraAWB: auto
  # auto-white-balance fixed gains. This can be used in place of rpiCameraAWB.
  # format: [red,blue]
  rpiCameraAWBGains: [0, 0]
  # denoise operating mode.
  # values: off, cdn_off, cdn_fast, cdn_hq
  rpiCameraDenoise: "off"
  # fixed shutter speed, in microseconds.
  rpiCameraShutter: 0
  # metering mode of the AEC/AGC algorithm.
  # values: centre, spot, matrix, custom
  rpiCameraMetering: centre
  # fixed gain
  rpiCameraGain: 0
  # EV compensation of the image [-10, 10]
  rpiCameraEV: 0
  # Region of interest, in format x,y,width,height
  rpiCameraROI:
  # whether to enable HDR on Raspberry Camera 3.
  rpiCameraHDR: false
  # tuning file
  rpiCameraTuningFile:
  # sensor mode, in format [width]:[height]:[bit-depth]:[packing]
  # bit-depth and packing are optional.
  rpiCameraMode:
  # frames per second
  rpiCameraFPS: 30
  # period between IDR frames
  rpiCameraIDRPeriod: 60
  # bitrate
  rpiCameraBitrate: 1000000
  # H264 profile
  rpiCameraProfile: main
  # H264 level
  rpiCameraLevel: '4.1'
  # Autofocus mode
  # values: auto, manual, continuous
  rpiCameraAfMode: continuous
  # Autofocus range
  # values: normal, macro, full
  rpiCameraAfRange: normal
  # Autofocus speed
  # values: normal, fast
  rpiCameraAfSpeed: normal
  # Lens position (for manual autofocus only), will be set to focus to a specific distance
  # calculated by the following formula: d = 1 / value
  # Examples: 0 moves the lens to infinity.
  #           0.5 moves the lens to focus on objects 2m away.
  #           2 moves the lens to focus on objects 50cm away.
  rpiCameraLensPosition: 0.0
  # Specifies the autofocus window, in the form x,y,width,height where the coordinates
  # are given as a proportion of the entire image.
  rpiCameraAfWindow:
  # enables printing text on each frame.
  rpiCameraTextOverlayEnable: false
  # text that is printed on each frame.
  # format is the one of the strftime() function.
  rpiCameraTextOverlay: '%Y-%m-%d %H:%M:%S - MediaMTX'

  ###############################################
  # Default path settings -> Hooks

  # Command to run when this path is initialized.
  # This can be used to publish a stream when the server is launched.
  # This is terminated with SIGINT when the program closes.
  # The following environment variables are available:
  # * MTX_PATH: path name
  # * RTSP_PORT: RTSP server port
  # * G1, G2, ...: regular expression groups, if path name is
  #   a regular expression.
  runOnInit:
  # Restart the command if it exits.
  runOnInitRestart: no

  # Command to run when this path is requested by a reader
  # and no one is publishing to this path yet.
  # This can be used to publish a stream on demand.
  # This is terminated with SIGINT when there are no readers anymore.
  # The following environment variables are available:
  # * MTX_PATH: path name
  # * MTX_QUERY: query parameters (passed by first reader)
  # * RTSP_PORT: RTSP server port
  # * G1, G2, ...: regular expression groups, if path name is
  #   a regular expression.
  runOnDemand:
  # Restart the command if it exits.
  runOnDemandRestart: no
  # Readers will be put on hold until the runOnDemand command starts publishing
  # or until this amount of time has passed.
  runOnDemandStartTimeout: 10s
  # The command will be closed when there are no
  # readers connected and this amount of time has passed.
  runOnDemandCloseAfter: 10s
  # Command to run when there are no readers anymore.
  # Environment variables are the same of runOnDemand.
  runOnUnDemand:

  # Command to run when the stream is ready to be read, whenever it is
  # published by a client or pulled from a server / camera.
  # This is terminated with SIGINT when the stream is not ready anymore.
  # The following environment variables are available:
  # * MTX_PATH: path name
  # * MTX_QUERY: query parameters (passed by publisher)
  # * RTSP_PORT: RTSP server port
  # * G1, G2, ...: regular expression groups, if path name is
  #   a regular expression.
  # * MTX_SOURCE_TYPE: source type
  # * MTX_SOURCE_ID: source ID
  runOnReady:
  # Restart the command if it exits.
  runOnReadyRestart: no
  # Command to run when the stream is not available anymore.
  # Environment variables are the same of runOnReady.
  runOnNotReady:

  # Command to run when a client starts reading.
  # This is terminated with SIGINT when a client stops reading.
  # The following environment variables are available:
  # * MTX_PATH: path name
  # * MTX_QUERY: query parameters (passed by reader)
  # * RTSP_PORT: RTSP server port
  # * G1, G2, ...: regular expression groups, if path name is
  #   a regular expression.
  # * MTX_READER_TYPE: reader type
  # * MTX_READER_ID: reader ID
  runOnRead:
  # Restart the command if it exits.
  runOnReadRestart: no
  # Command to run when a client stops reading.
  # Environment variables are the same of runOnRead.
  runOnUnread:

  # Command to run when a recording segment is created.
  # The following environment variables are available:
  # * MTX_PATH: path name
  # * RTSP_PORT: RTSP server port
  # * G1, G2, ...: regular expression groups, if path name is
  #   a regular expression.
  # * MTX_SEGMENT_PATH: segment file path
  runOnRecordSegmentCreate:

  # Command to run when a recording segment is complete.
  # The following environment variables are available:
  # * MTX_PATH: path name
  # * RTSP_PORT: RTSP server port
  # * G1, G2, ...: regular expression groups, if path name is
  #   a regular expression.
  # * MTX_SEGMENT_PATH: segment file path
  runOnRecordSegmentComplete:

###############################################
# Path settings

# Settings in "paths" are applied to specific paths, and the map key
# is the name of the path.
# Any setting in "pathDefaults" can be overridden here.
# It's possible to use regular expressions by using a tilde as prefix,
# for example "~^(test1|test2)$" will match both "test1" and "test2",
# for example "~^prefix" will match all paths that start with "prefix".
paths:
  # example:
  # my_camera:
  #   source: rtsp://my_camera

  # Settings under path "all_others" are applied to all paths that
  # do not match another entry.
  all_others:
`
