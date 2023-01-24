package backend

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func concatIfNotEmpty(str string, add string) string {
	if str != "" {
		return str + add
	}
	return str
}

func GenerateServerConfig(state *State, w io.Writer) error {
	PostUp := ""
	PostDown := ""

	wd, err := os.Getwd()
	if err != nil {
		panic("can't os.Getwd " + err.Error())
	}
	comment := "# This file is generated by wg-dir-conf from directory " + wd
	comment += "\n# It it likely to be overwritten.\n"

	_, err = fmt.Fprintf(w, "%s\n[Interface]\n", comment)
	if err != nil {
		return fmt.Errorf("generateServerConfig error %w", err)
	}
	if state.Server.Address4 != "" {
		_, _ = fmt.Fprintln(w, "Address =", state.Server.Address4)
		PostUp = strings.TrimRight(state.Server.PostUp4, " ;")
		PostDown = strings.TrimRight(state.Server.PostDown4, "; ")
	}
	if state.Server.Address6 != "" {
		_, _ = fmt.Fprintln(w, "Address =", state.Server.Address6)
		PostUp = concatIfNotEmpty(PostUp, "; ")
		PostDown = concatIfNotEmpty(PostDown, "; ")

		PostUp = PostUp + state.Server.PostUp6
		PostDown = PostDown + state.Server.PostDown6
	}
	_, _ = fmt.Fprintln(w, "PostUp =", PostUp)
	_, _ = fmt.Fprintln(w, "PostDown =", PostDown)
	_, _ = fmt.Fprintln(w, "ListenPort =", state.Server.ListenPort)
	_, _ = fmt.Fprintln(w, "PrivateKey =", state.Server.PrivateKey)

	for _, client := range state.Clients {
		err = generateServerPeerConfig(state.Server, client, w)
		if err != nil {
			return fmt.Errorf("generateServerConfig error %w", err)
		}
	}

	return nil
}

func generateServerPeerConfig(srv *Server, client *Client, w io.Writer) error {
	_, err := fmt.Fprintf(w, "\n# peer %s\n", client.name)
	if err != nil {
		return fmt.Errorf("generateServerConfig error %w", err)
	}
	_, _ = fmt.Fprintln(w, "[Peer]")
	if srv.PresharedKey != "" {
		_, _ = fmt.Fprintln(w, "PresharedKey =", srv.PresharedKey)
	}
	_, _ = fmt.Fprintln(w, "PublicKey =", client.PublicKey)
	_, _ = fmt.Fprintln(w, "AllowedIPs =", client.AllowedIps(srv))

	return nil
}

func GenerateClientConfig(server *Server, client *Client, w io.Writer) error {
	_, err := fmt.Fprintf(w, "[Interface]\n")
	if err != nil {
		return fmt.Errorf("GenerateClientConfig error %w", err)
	}
	_, _ = fmt.Fprintln(w, "PrivateKey =", client.PrivateKey)
	_, _ = fmt.Fprintln(w, "Address =", client.AllowedIps(server))

	if server.ClientDNS != "" {
		_, _ = fmt.Fprintln(w, "DNS =", server.ClientDNS)
	}

	_, _ = fmt.Fprintln(w, "\n[Peer]")
	if server.PresharedKey != "" {
		_, _ = fmt.Fprintln(w, "PresharedKey =", server.PresharedKey)
	}
	_, _ = fmt.Fprintln(w, "PublicKey =", server.PublicKey)
	_, _ = fmt.Fprintln(w, "AllowedIPs =", server.ClientRoute)
	_, _ = fmt.Fprintln(w, "Endpoint =", server.ClientServerEndpoint+":"+strconv.Itoa(int(server.ListenPort)))

	if server.ClientPersistentKeepalive != 0 {
		_, _ = fmt.Fprintln(w, "PersistentKeepalive =", server.ClientPersistentKeepalive)
	}

	return nil
}
