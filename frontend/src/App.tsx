import { useEffect, useState } from 'react';
import { ConfigProvider, theme, Layout, Row, Col, Card, Statistic, Table, Button, Progress, Tag, message, Typography } from 'antd';
import { CloudServerOutlined, ReloadOutlined, ClearOutlined, FileTextOutlined, DatabaseOutlined, PieChartOutlined, CheckCircleOutlined } from '@ant-design/icons';
import axios from 'axios';

const { Header, Content } = Layout;
const { Title, Text } = Typography;

const API_BASE = '/api';

interface StatsData {
  total_files: number;
  total_chunks: number;
  unique_chunks: number;
  logical_size: number;
  physical_size: number;
  dedup_ratio_percent: string;
}

interface FileData {
  root_hash: string;
  file_name: string;
  file_size: number;
  created_at: string;
}

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

export default function App() {
  const [stats, setStats] = useState<StatsData>({
    total_files: 2,
    total_chunks: 1970,
    unique_chunks: 986,
    logical_size: 109051904, // 104 MB
    physical_size: 54525952,  // 52 MB
    dedup_ratio_percent: '50.00'
  });
  const [files, setFiles] = useState<FileData[]>([
    { root_hash: '9894e48b2a53823481239abcef12389172346123846182364812634817263481', file_name: 'v2_modified.dat', file_size: 54525952, created_at: '2026-06-28 10:10:45' },
    { root_hash: 'c37ad13cf2d61982736481726348172634817263481726348172634817263481', file_name: 'v1_original.dat', file_size: 54525952, created_at: '2026-06-28 10:10:42' }
  ]);
  const [loading, setLoading] = useState(false);
  const [isConnected, setIsConnected] = useState(false);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [resStats, resFiles] = await Promise.all([
        axios.get(`${API_BASE}/stats`),
        axios.get(`${API_BASE}/files`)
      ]);
      if (resStats.data && resStats.data.data) {
        setStats(resStats.data.data);
      }
      if (resFiles.data && resFiles.data.data) {
        setFiles(resFiles.data.data);
      }
      setIsConnected(true);
      message.success('已同步最新引擎去重统计数据');
    } catch (err) {
      setIsConnected(false);
      // 后端未运行则静默使用 mock 演示数据
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleGC = async () => {
    try {
      await axios.post(`${API_BASE}/gc`);
      message.success('垃圾回收完成，无引用数据块已清理！');
      fetchData();
    } catch (err) {
      message.info('演示模式：模拟垃圾回收成功！');
    }
  };

  const columns = [
    {
      title: '文件名称',
      dataIndex: 'file_name',
      key: 'file_name',
      render: (text: string) => <Tag color="blue" icon={<FileTextOutlined />}>{text}</Tag>
    },
    {
      title: '逻辑文件大小',
      dataIndex: 'file_size',
      key: 'file_size',
      render: (val: number) => <Text strong style={{ color: '#52c41a' }}>{formatBytes(val)}</Text>
    },
    {
      title: 'CAS 根指纹 (Root Hash)',
      dataIndex: 'root_hash',
      key: 'root_hash',
      render: (hash: string) => (
        <Text copyable={{ text: hash }} style={{ fontFamily: 'monospace', color: '#8b949e' }}>
          {hash.substring(0, 16)}...{hash.substring(hash.length - 8)}
        </Text>
      )
    },
    {
      title: '存储时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (val: string) => {
        try {
          const d = new Date(val);
          if (isNaN(d.getTime())) return <Text type="secondary">{val}</Text>;
          const formatted = d.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            hour12: false
          }).replace(/\//g, '-');
          return <Text type="secondary">{formatted}</Text>;
        } catch {
          return <Text type="secondary">{val}</Text>;
        }
      }
    }
  ];

  return (
    <ConfigProvider theme={{ algorithm: theme.darkAlgorithm, token: { colorPrimary: '#1677ff', borderRadius: 12 } }}>
      <div className="app-background" />
      <Layout style={{ background: 'transparent', minHeight: '100vh' }}>
        <Header style={{ background: 'rgba(13, 17, 23, 0.8)', backdropFilter: 'blur(20px)', borderBottom: '1px solid rgba(255,255,255,0.08)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 48px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <CloudServerOutlined style={{ fontSize: 28, color: '#1677ff' }} />
            <span style={{ fontSize: 22, fontWeight: 700 }} className="gradient-text">CDC Dedup Engine</span>
          </div>
          <div>
            <Tag color={isConnected ? 'success' : 'warning'} icon={<CheckCircleOutlined />}>
              {isConnected ? 'Go Backend Live Connected' : 'Demo Dashboard Mode'}
            </Tag>
          </div>
        </Header>

        <Content style={{ padding: '48px', maxWidth: 1400, margin: '0 auto', width: '100%' }}>
          <div style={{ marginBottom: 36, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <Title level={2} style={{ margin: 0 }}>分布式大文件去重引擎看板</Title>
              <Text type="secondary">基于 Content-Defined Chunking (CDC) 与内容寻址 (CAS) 的增量去重监控</Text>
            </div>
            <div style={{ display: 'flex', gap: 12 }}>
              <Button type="primary" icon={<ReloadOutlined />} loading={loading} onClick={fetchData} size="large">
                刷新数据
              </Button>
              <Button danger icon={<ClearOutlined />} onClick={handleGC} size="large">
                执行 GC 垃圾回收
              </Button>
            </div>
          </div>

          <Row gutter={[24, 24]}>
            <Col xs={24} sm={12} lg={6}>
              <Card className="glass-card" bordered={false}>
                <Statistic title={<span style={{ color: '#8b949e' }}><FileTextOutlined /> 托管文件总数</span>} value={stats.total_files} valueStyle={{ color: '#4096ff', fontSize: 32, fontWeight: 700 }} suffix="个文件" />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card className="glass-card" bordered={false}>
                <Statistic title={<span style={{ color: '#8b949e' }}><DatabaseOutlined /> 逻辑总数据大小</span>} value={formatBytes(stats.logical_size)} valueStyle={{ color: '#a855f7', fontSize: 32, fontWeight: 700 }} />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card className="glass-card" bordered={false}>
                <Statistic title={<span style={{ color: '#8b949e' }}><CloudServerOutlined /> 物理实际占用空间</span>} value={formatBytes(stats.physical_size)} valueStyle={{ color: '#52c41a', fontSize: 32, fontWeight: 700 }} />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card className="glass-card" bordered={false}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Statistic title={<span style={{ color: '#8b949e' }}><PieChartOutlined /> 全局空间去重率</span>} value={stats.dedup_ratio_percent} suffix="%" valueStyle={{ color: '#faad14', fontSize: 32, fontWeight: 700 }} />
                  <Progress type="circle" percent={parseFloat(stats.dedup_ratio_percent)} width={64} strokeColor={{ '0%': '#108ee9', '100%': '#87d068' }} />
                </div>
              </Card>
            </Col>
          </Row>

          <Row gutter={[24, 24]} style={{ marginTop: 24 }}>
            <Col span={24}>
              <Card className="glass-card" bordered={false} title={<span style={{ fontSize: 18, fontWeight: 600 }}>🧱 数据块去重状态对比</span>}>
                <Row gutter={24} style={{ textAlign: 'center', padding: '16px 0' }}>
                  <Col span={12}>
                    <Statistic title="系统总切块引用数 (Total Chunk References)" value={stats.total_chunks} valueStyle={{ color: '#f0f6fc', fontSize: 28 }} />
                    <Text type="secondary" style={{ fontSize: 12 }}>如果不去重，系统需要存储的物理分块数量</Text>
                  </Col>
                  <Col span={12}>
                    <Statistic title="实际存储唯一指纹块 (Unique Chunks Stored)" value={stats.unique_chunks} valueStyle={{ color: '#52c41a', fontSize: 28 }} />
                    <Text type="secondary" style={{ fontSize: 12 }}>通过 SHA-256 内容寻址去重后实际写入磁盘的数据块</Text>
                  </Col>
                </Row>
              </Card>
            </Col>
          </Row>

          <Row gutter={[24, 24]} style={{ marginTop: 24 }}>
            <Col span={24}>
              <Card className="glass-card" bordered={false} title={<span style={{ fontSize: 18, fontWeight: 600 }}>🗂️ 托管文件版本映射列表</span>}>
                <Table columns={columns} dataSource={files} rowKey="root_hash" pagination={false} />
              </Card>
            </Col>
          </Row>
        </Content>
      </Layout>
    </ConfigProvider>
  );
}
