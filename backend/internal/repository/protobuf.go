package repository

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func decodeRepoIndex(b []byte) (*repoIndex, error) {
	idx := &repoIndex{}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, fmt.Errorf("consume tag: %w", protowire.ParseError(n))
		}
		b = b[n:]

		switch num {
		case 1, 2, 3, 102:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, fmt.Errorf("consume string field %d: %w", num, protowire.ParseError(n))
			}
			b = b[n:]
			switch num {
			case 1:
				idx.Name = string(v)
			case 2:
				idx.BadgeLabel = string(v)
			case 3:
				idx.SigningKey = string(v)
			case 102:
				idx.ExtensionListURL = string(v)
			}
		case 4:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, fmt.Errorf("consume contact: %w", protowire.ParseError(n))
			}
			b = b[n:]
			c, err := decodeContact(v)
			if err != nil {
				return nil, err
			}
			idx.Contact = *c
		case 101:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, fmt.Errorf("consume extension_list: %w", protowire.ParseError(n))
			}
			b = b[n:]
			el, err := decodeExtensionList(v)
			if err != nil {
				return nil, err
			}
			idx.ExtensionList = el
		default:
			n, err := skipField(b, typ)
			if err != nil {
				return nil, err
			}
			b = b[n:]
		}
	}
	return idx, nil
}

func decodeContact(b []byte) (*contact, error) {
	c := &contact{}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, fmt.Errorf("consume tag: %w", protowire.ParseError(n))
		}
		b = b[n:]
		switch num {
		case 1, 2:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, fmt.Errorf("consume contact field %d: %w", num, protowire.ParseError(n))
			}
			b = b[n:]
			if num == 1 {
				c.Website = string(v)
			} else {
				c.Discord = string(v)
			}
		default:
			n, err := skipField(b, typ)
			if err != nil {
				return nil, err
			}
			b = b[n:]
		}
	}
	return c, nil
}

func decodeExtensionList(b []byte) (*extensionList, error) {
	el := &extensionList{}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, fmt.Errorf("consume tag: %w", protowire.ParseError(n))
		}
		b = b[n:]
		if num == 1 {
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, fmt.Errorf("consume extension entry: %w", protowire.ParseError(n))
			}
			b = b[n:]
			ext, err := decodeExtension(v)
			if err != nil {
				return nil, err
			}
			el.Extensions = append(el.Extensions, *ext)
			continue
		}
		n, err := skipField(b, typ)
		if err != nil {
			return nil, err
		}
		b = b[n:]
	}
	return el, nil
}

func decodeExtension(b []byte) (*extension, error) {
	ext := &extension{}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, fmt.Errorf("consume tag: %w", protowire.ParseError(n))
		}
		b = b[n:]
		switch num {
		case 1, 2, 4, 6:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, fmt.Errorf("consume extension field %d: %w", num, protowire.ParseError(n))
			}
			b = b[n:]
			switch num {
			case 1:
				ext.Name = string(v)
			case 2:
				ext.PackageName = string(v)
			case 4:
				ext.ExtensionLib = string(v)
			case 6:
				ext.VersionName = string(v)
			}
		case 3:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, fmt.Errorf("consume resources: %w", protowire.ParseError(n))
			}
			b = b[n:]
			res, err := decodeResources(v)
			if err != nil {
				return nil, err
			}
			ext.Resources = *res
		case 5:
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return nil, fmt.Errorf("consume version_code: %w", protowire.ParseError(n))
			}
			b = b[n:]
			ext.VersionCode = int64(v)
		case 7:
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return nil, fmt.Errorf("consume content_warning: %w", protowire.ParseError(n))
			}
			b = b[n:]
			ext.ContentWarning = int32(v)
		case 8:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, fmt.Errorf("consume source entry: %w", protowire.ParseError(n))
			}
			b = b[n:]
			s, err := decodeSource(v)
			if err != nil {
				return nil, err
			}
			ext.Sources = append(ext.Sources, *s)
		default:
			n, err := skipField(b, typ)
			if err != nil {
				return nil, err
			}
			b = b[n:]
		}
	}
	return ext, nil
}

func decodeResources(b []byte) (*resources, error) {
	r := &resources{}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, fmt.Errorf("consume tag: %w", protowire.ParseError(n))
		}
		b = b[n:]
		switch num {
		case 1, 2, 501:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, fmt.Errorf("consume resources field %d: %w", num, protowire.ParseError(n))
			}
			b = b[n:]
			switch num {
			case 1:
				r.ApkURL = string(v)
			case 2:
				r.IconURL = string(v)
			case 501:
				r.JarURL = string(v)
			}
		default:
			n, err := skipField(b, typ)
			if err != nil {
				return nil, err
			}
			b = b[n:]
		}
	}
	return r, nil
}

func decodeSource(b []byte) (*source, error) {
	s := &source{}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, fmt.Errorf("consume tag: %w", protowire.ParseError(n))
		}
		b = b[n:]
		switch num {
		case 1:
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return nil, fmt.Errorf("consume source id: %w", protowire.ParseError(n))
			}
			b = b[n:]
			s.ID = int64(v)
		case 2, 3, 4, 7:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, fmt.Errorf("consume source field %d: %w", num, protowire.ParseError(n))
			}
			b = b[n:]
			switch num {
			case 2:
				s.Name = string(v)
			case 3:
				s.Language = string(v)
			case 4:
				s.HomeURL = string(v)
			case 7:
				s.Message = string(v)
			}
		case 5:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, fmt.Errorf("consume mirror_url: %w", protowire.ParseError(n))
			}
			b = b[n:]
			s.MirrorURLs = append(s.MirrorURLs, string(v))
		default:
			n, err := skipField(b, typ)
			if err != nil {
				return nil, err
			}
			b = b[n:]
		}
	}
	return s, nil
}

func skipField(b []byte, typ protowire.Type) (int, error) {
	n := protowire.ConsumeFieldValue(0, typ, b)
	if n < 0 {
		return 0, fmt.Errorf("skip field: %w", protowire.ParseError(n))
	}
	return n, nil
}
