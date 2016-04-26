<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xsl">
  <xsl:output encoding="ASCII" omit-xml-declaration="yes" indent="yes"/>
  <xsl:param name="XsltParamAttributeValue" />
  <xsl:param name="XsltParamInnerTextValue" />

  <!-- Match and copy everything -->
  <xsl:template match="@*|node()">
    <xsl:copy>
      <xsl:apply-templates select="node()|@*"/>
    </xsl:copy>
  </xsl:template>

  <!-- replace the attribute value -->
  <xsl:template match="NodeLevel1/NodeLevel2/NodeLevel3/@Name">
    <xsl:attribute name="Name">
      <xsl:value-of select="$XsltParamAttributeValue"/>
    </xsl:attribute>
  </xsl:template>

  <!-- replace the inner text value -->
  <xsl:template match="NodeLevel1/NodeLevel2/NodeLevel3/text()">
    <xsl:value-of select="$XsltParamInnerTextValue"/>
  </xsl:template>
</xsl:stylesheet>