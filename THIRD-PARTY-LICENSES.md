# Third-Party Licenses

This document lists all third-party dependencies and their licenses used in Sibyl.

## Critical License Dependencies

### MuPDF (AGPL v3) ⚠️

**Package**: `github.com/gen2brain/go-fitz`  
**Underlying Library**: MuPDF  
**License**: GNU Affero General Public License v3.0  
**Licensor**: Artifex Software, Inc.  
**Website**: https://mupdf.com/licensing/

**⚠️ IMPORTANT**: This dependency requires your entire application to be licensed under AGPL v3 when used in network services or distributed software.

**License Text**: https://www.gnu.org/licenses/agpl-3.0.html

**Commercial Licensing**: Available from Artifex Software for proprietary use.

---

## Permissive License Dependencies

The following dependencies use permissive open-source licenses that do not impose restrictions on your application's licensing:

### MCP Go SDK
**Package**: `github.com/mark3labs/mcp-go`  
**License**: MIT License  
**Repository**: https://github.com/mark3labs/mcp-go

### Environment Variable Loading
**Package**: `github.com/joho/godotenv`  
**License**: MIT License  
**Repository**: https://github.com/joho/godotenv

### Google APIs
**Package**: `google.golang.org/api`  
**License**: Apache License 2.0  
**Repository**: https://github.com/googleapis/google-api-go-client

---

## Complete Dependency List

### Direct Dependencies
```
github.com/gen2brain/go-fitz v1.24.15         - AGPL v3 (via MuPDF)
github.com/joho/godotenv v1.5.1               - MIT License
github.com/mark3labs/mcp-go v0.34.0           - MIT License
google.golang.org/api v0.241.0                - Apache 2.0
```

### Indirect Dependencies (Partial List)
```
cloud.google.com/go/auth v0.16.2              - Apache 2.0
github.com/google/uuid v1.6.0                 - BSD 3-Clause
golang.org/x/crypto v0.39.0                   - BSD 3-Clause
golang.org/x/oauth2 v0.30.0                   - BSD 3-Clause
google.golang.org/grpc v1.73.0                - Apache 2.0
```

---

## License Compatibility Matrix

| Your Application License | Compatible | Notes |
|--------------------------|------------|-------|
| AGPL v3                  | ✅ Yes     | Fully compatible |
| GPL v3                   | ✅ Yes     | AGPL is compatible with GPL v3 |
| LGPL                     | ❌ No      | AGPL is more restrictive |
| Apache 2.0               | ❌ No      | AGPL viral licensing applies |
| MIT                      | ❌ No      | AGPL viral licensing applies |
| BSD                      | ❌ No      | AGPL viral licensing applies |
| Proprietary/Commercial   | ❌ No      | Requires commercial MuPDF license |

---

## Compliance Checklist

### For AGPL v3 Compliance
- [ ] License your entire application under AGPL v3
- [ ] Provide complete source code to all users
- [ ] Include AGPL v3 license text in distribution
- [ ] Ensure users can rebuild from source
- [ ] Provide prominent notice of AGPL licensing

### For Commercial Use
- [ ] Contact Artifex Software for MuPDF commercial license
- [ ] Ensure commercial license covers your use case
- [ ] Update your application's license accordingly
- [ ] Remove AGPL notices for MuPDF portion

---

## Getting Help

### Legal Questions
For questions about license compliance, consult with a qualified legal professional familiar with open-source licensing.

### Commercial Licensing
**Artifex Software, Inc.**  
- Website: https://mupdf.com/licensing/
- Email: Contact through their website
- Product: MuPDF commercial licenses

### Technical Questions
- GitHub Issues: https://github.com/your-username/sibyl/issues
- Documentation: See README.md

---

## License Change History

- **v1.0.0**: Initial release with Apache 2.0 + AGPL v3 dependency notice
- Future versions will document any licensing changes here

---

*Last updated: [Current Date]*  
*For the most current license information, check the latest version of this file.*